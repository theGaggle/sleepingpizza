package websockets

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/bakape/meguca/db"
	"github.com/bakape/meguca/parser"
	"github.com/bakape/meguca/types"
	"github.com/bakape/meguca/util"
	r "github.com/dancannon/gorethink"
)

var (
	errNoPostOpen          = errors.New("no post open")
	errLineEmpty           = errors.New("line empty")
	errInvalidSpliceCoords = errors.New("invalid splice coordinates")
	errSpliceTooLong       = errors.New("splice text too long")
	errSpliceNOOP          = errors.New("splice NOOP")
	errTextOnly            = errors.New("text only board")
	errHasImage            = errors.New("post already has image")
)

// Request or to replace the current line's text starting at an exact position
// in the current line
type spliceRequest struct {
	Start int    `json:"start"`
	Len   int    `json:"len"`
	Text  string `json:"text"`
}

// Response to a spliceRequest. Sent to all listening clients.
type spliceResponse struct {
	ID int64 `json:"id"`
	spliceRequest
}

// Message sent to listening clients about a link or backlink insertion into
// a post
type linkMessage struct {
	ID    int64         `json:"id"`
	Links types.LinkMap `json:"links"`
}

// Message sent to all clients to inject a command result into a model
type commandMessage struct {
	ID int64 `json:"id"`
	types.Command
}

// Message that signals and insertion of an image into an existing post
type imageMessage struct {
	types.Image
	ID int64 `json:"id"`
}

// Append a rune to the body of the open post
func appendRune(data []byte, c *Client) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	if c.openPost.bodyLength+1 > parser.MaxLengthBody {
		return parser.ErrBodyTooLong
	}
	var char rune
	if err := decodeMessage(data, &char); err != nil {
		return err
	}

	if char == '\n' {
		return parseLine(c, true)
	}

	id := c.openPost.id
	msg, err := EncodeMessage(MessageAppend, [2]int64{id, int64(char)})
	if err != nil {
		return err
	}

	update := map[string]r.Term{
		"body": postBody(id).Add(string(char)),
	}
	if err := c.updatePost(update, msg); err != nil {
		return err
	}
	c.openPost.WriteRune(char)
	c.openPost.bodyLength++

	return nil
}

// Shorthand for retrievinf the post's body field from a thread document
func postBody(id int64) r.Term {
	return r.Row.
		Field("posts").
		Field(util.IDToString(id)).
		Field("body")
}

// Helper for running post update queries on the current open post
func (c *Client) updatePost(update interface{}, msg []byte) error {
	q := r.
		Table("threads").
		Get(c.openPost.op).
		Update(CreateUpdate(c.openPost.id, update, msg))
	return db.Write(q)
}

// CreateUpdate is a helper for creating post update maps
func CreateUpdate(
	id int64,
	update interface{},
	msg []byte,
) map[string]interface{} {
	return map[string]interface{}{
		"log": appendLog(msg),
		"posts": map[string]interface{}{
			util.IDToString(id): update,
		},
	}
}

// Shorthand for creating a replication log append query
func appendLog(msg []byte) r.Term {
	return r.Row.Field("log").Append(msg)
}

// Parse line contents and commit newline. If line contains hash commands or
// links to other posts also commit those and generate backlinks, if needed.
// Appending the newline can be optionally omitted, to optimise post closing
// and similar.
func parseLine(c *Client, insertNewline bool) error {
	c.openPost.bodyLength++
	links, comm, err := parser.ParseLine(c.openPost.Bytes(), c.openPost.board)
	if err != nil {
		return err
	}
	defer c.openPost.Reset()
	idStr := util.IDToString(c.openPost.id)

	switch {
	case comm.Val != nil:
		err = writeCommand(comm, idStr, c)
	case links != nil:
		err = writeLinks(links, c)
	}
	if err != nil {
		return err
	}

	if insertNewline {
		msg, err := EncodeMessage(MessageAppend, [2]int64{
			c.openPost.id,
			int64('\n'),
		})
		if err != nil {
			return err
		}
		update := map[string]r.Term{
			"body": r.Row.
				Field("posts").
				Field(idStr).
				Field("body").
				Add("\n"),
		}
		if err := c.updatePost(update, msg); err != nil {
			return err
		}
	}

	return nil
}

// Write a hash command to the database
func writeCommand(comm types.Command, idStr string, c *Client) error {
	msg, err := EncodeMessage(MessageCommand, commandMessage{
		ID:      c.openPost.id,
		Command: comm,
	})
	if err != nil {
		return err
	}
	update := map[string]r.Term{
		"commands": r.Row.
			Field("posts").
			Field(idStr).
			Field("commands").
			Default([]types.Command{}).
			Append(comm),
	}
	return c.updatePost(update, msg)
}

// Write new links to other posts to the database
func writeLinks(links types.LinkMap, c *Client) error {
	msg, err := EncodeMessage(MessageLink, linkMessage{
		ID:    c.openPost.id,
		Links: links,
	})
	if err != nil {
		return err
	}
	update := map[string]types.LinkMap{
		"links": links,
	}
	if err := c.updatePost(update, msg); err != nil {
		return err
	}

	// Most often this loop will iterate only once, so no need to think heavily
	// about optimisations
	for destID := range links {
		id := c.openPost.id
		op := c.openPost.op
		board := c.openPost.board
		if err := writeBacklink(id, op, board, destID); err != nil {
			return err
		}
	}

	return nil
}

// Writes the location data of the post linking a post to the the post being
// linked
func writeBacklink(id, op int64, board string, destID int64) error {
	msg, err := EncodeMessage(MessageBacklink, linkMessage{
		ID: destID,
		Links: types.LinkMap{
			id: {
				OP:    op,
				Board: board,
			},
		},
	})
	if err != nil {
		return err
	}

	update := map[string]map[string]types.Link{
		"backlinks": {
			util.IDToString(id): types.Link{
				OP:    op,
				Board: board,
			},
		},
	}
	q := r.
		Table("threads").
		GetAllByIndex("post", destID).
		Update(CreateUpdate(destID, update, msg))

	return db.Write(q)
}

// Remove one character from the end of the line in the open post
func backspace(_ []byte, c *Client) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	length := c.openPost.Len()
	if length == 0 {
		return errLineEmpty
	}
	_, lastRuneLen := utf8.DecodeLastRune(c.openPost.Bytes())
	c.openPost.Truncate(length - lastRuneLen)
	c.openPost.bodyLength--

	id := c.openPost.id
	update := map[string]r.Term{
		"body": postBody(id).Slice(0, -1),
	}
	msg, err := EncodeMessage(MessageBackspace, id)
	if err != nil {
		return err
	}
	return c.updatePost(update, msg)
}

// Close an open post and parse the last line, if needed.
func closePost(_ []byte, c *Client) error {
	if c.openPost.id == 0 {
		return errNoPostOpen
	}
	if c.openPost.Len() != 0 {
		if err := parseLine(c, false); err != nil {
			return err
		}
	}

	update := map[string]bool{
		"editing": false,
	}
	msg, err := EncodeMessage(MessageClosePost, c.openPost.id)
	if err != nil {
		return err
	}
	if err := c.updatePost(update, msg); err != nil {
		return err
	}

	c.openPost = openPost{}
	return nil
}

// Splice the current line's text in the open post. This call is also used for
// text pastes.
func spliceText(data []byte, c *Client) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}

	var req spliceRequest
	oldLength := len(c.openPost.String())
	err := decodeMessage(data, &req)
	switch {
	case err != nil:
		return err
	case req.Start < 0, req.Start+req.Len > oldLength:
		return errInvalidSpliceCoords
	case req.Len == 0 && req.Text == "":
		return errSpliceNOOP // This does nothing. Client-side error.
	case len(req.Text) > parser.MaxLengthBody:
		return errSpliceTooLong // Nice try, kid
	}

	return spliceLine(req, c)
}

// Splice the first line of the text. If there are more lines, parse the
// previous one and recurse until all lines are parsed.
func spliceLine(req spliceRequest, c *Client) error {
	old := c.openPost.String()
	start := old[:req.Start]
	var end string

	// -1 has special meaning - slice off till line end.
	if req.Len < 0 {
		end = req.Text
		c.openPost.bodyLength += len(req.Text) - len(old[req.Start:])
	} else {
		end = req.Text + old[req.Start+req.Len:]
		c.openPost.bodyLength += -req.Len + len(req.Text)
	}
	res := spliceResponse{
		ID:            c.openPost.id,
		spliceRequest: req,
	}

	// Slice until newline, if any, and delay the next line's splicing until the
	// next recursive spliceLine call
	var delayed string
	if firstNewline := strings.IndexRune(end, '\n'); firstNewline > -1 {
		delayed = end[firstNewline+1:]
		end = end[:firstNewline]
		res.Len = -1 // Special meaning. Client should replace till line end.
		res.Text = end
		c.openPost.bodyLength -= len(delayed) + 1
	}

	// Goes over max post length. Trim the end.
	exceeding := c.openPost.bodyLength - parser.MaxLengthBody
	if exceeding > 0 {
		end = end[:len(end)-exceeding]
		res.Len = -1
		res.Text = end
		c.openPost.bodyLength = parser.MaxLengthBody
	}

	c.openPost.Reset()
	new := start + end
	c.openPost.WriteString(new)

	msg, err := EncodeMessage(MessageSplice, res)
	if err != nil {
		return err
	}

	// Split body into lines, remove last line and replace with new text
	update := map[string]r.Term{
		"body": postBody(c.openPost.id).
			Split("\n").
			Do(func(b r.Term) r.Term {
				return b.
					Slice(0, -1).
					Append(new).
					Fold("", func(all, line r.Term) r.Term {
						return all.Add(
							all.Eq("").Branch(
								line,
								r.Expr("\n").Add(line),
							),
						)
					})
			}),
	}
	if err := c.updatePost(update, msg); err != nil {
		return err
	}

	// Recurse on the text sliced after the newline, until all lines are
	// processed. Unless the text body is already over max length
	if delayed != "" && exceeding < 0 {
		if err := parseLine(c, true); err != nil {
			return err
		}
		return spliceLine(spliceRequest{Text: delayed}, c)
	}
	return nil
}

// Insert and image into an existing open post
func insertImage(data []byte, c *Client) error {
	if has, err := c.hasPost(); err != nil {
		return err
	} else if !has {
		return nil
	}
	if c.openPost.hasImage {
		return errHasImage
	}

	var req imageRequest
	if err := decodeMessage(data, &req); err != nil {
		return err
	}

	var textOnly bool
	q := db.GetBoardConfig(c.openPost.board).Field("textOnly")
	if err := db.One(q, &textOnly); err != nil {
		return err
	}
	if textOnly {
		return errTextOnly
	}

	img, err := getImage(req.Token, req.Name, req.Spoiler)
	if err != nil {
		return err
	}
	msg, err := EncodeMessage(MessageInsertImage, imageMessage{
		ID:    c.openPost.id,
		Image: *img,
	})
	if err != nil {
		return err
	}
	update := map[string]interface{}{
		"log":      appendLog(msg),
		"imageCtr": r.Row.Field("imageCtr").Add(1),
		"posts": map[string]map[string]types.Image{
			util.IDToString(c.openPost.id): {
				"image": *img,
			},
		},
	}
	q = r.Table("threads").Get(c.openPost.op).Update(update)
	if err := db.Write(q); err != nil {
		return err
	}

	c.openPost.hasImage = true
	return nil
}
