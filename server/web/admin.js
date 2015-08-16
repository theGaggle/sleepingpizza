/*
Serve moderation script
 */

let _ = require('underscore'),
	caps = require('../caps'),
	express = require('express'),
	resources = require('../state').resources,
	util = require('./util');

let router = module.exports = express.Router();

let headers = _.clone(util.noCacheHeaders);
headers['Content-Type'] = 'text/javascript; charset=UTF-8';

router.get('/mod.js', function (req, res) {
	// Admin/Moderator privelege is injected on page render and verified
	// serverside. Thus, we can serve the same bundle for both admins and mods.
	if (!caps.checkAuth('janitor', req.ident))
		return res.sendStatus(404);

	const modJS = resources.modJs;
	if (!modJS)
		return res.sendStatus(500);

	// Not hosted as a file to prevent unauthorised access
	res.set(headers);
	res.send(modJS);
});

router.get('/mod.js.map', function (req, res) {
	res.send(resources.modSourcemap);
});
