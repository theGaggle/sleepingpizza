Real-time imageboard.
MIT licensed.

**NOTE: The git master branch contains the development version of the board.
Crashes are not uncommon as new features are added and patched. The database 
structure and transport API are in active development. Don't use the master 
branch in production, unless you are comfortable with flushing redis and 
reconfiguring several times a month. For more stable revisions, please 
[download the latest release](https://github.com/bakape/meguca/releases).**

##Setup
* Install dependencies listed below
* Sign up for reCAPTCHA
* Run `npm install` to install npm deps and compile C++ addons
* Configure the files in `config/`
* Run `node builder.js` to satrt an auto-reloading development server

##Production
* Have your webserver serve `www/`
  * Some features will not work without a dedicated webserver. It is highly
  recommended not to use the the default inbuilt webserver in production.
* Run `node server/server.js` to start the server
* You can update `config/hot.js` on-the-fly with `node server/kill.js`
* To remove compiled server dependancies run `make clean`
* Similarly `make client_clean` removes compiled client files

##Updating
* To recompile client JS & CSS run `make client`. The new files can be loaded
 into a running server with `node server/kill.js`
* After upgrading an io.js version or a meguca release run `make upgrade` to 
recompile all dependancies

##Dependencies
* ImageMagick
* gcc, g++, make
* libpng with development headers
* [io.js](https://iojs.org) >=2.0.0
* redis

###Optional dependencies for various features
* ffmpeg 2.2+ for WebM support
  * with libmp3lame for MP3
* pngquant  2.3.0+ for PNG thumbnails

###Optional npm dependancies
* node-expat for icecast now-playing banners

## Documentation
* [JSON API: docs/api.md](https://github.com/bakape/meguca/blob/master/docs/api.md)
* [init script example: docs/doushio.initscript.example](https://github.com/bakape/meguca/blob/master/docs/doushio.initscript.example)
* [ngingx configuration example: docs/nginx.config.example](https://github.com/bakape/meguca/blob/master/docs/nginx.conf.example)

##Standalone upkeep scripts
* scripts/backup.js - uploads rdb to S3
* scripts/backup.sh - MEGA backup script
* scripts/purge_bans.sh - Removes all bans
* scripts/purge_mod_sessions.sh - Closes all active moderator sessions
* scripts/radio.js - icecast2 server integration
