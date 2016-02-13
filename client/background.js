/*
 * Background controller. Wallpapers, proper fitting and video backgrounds
 */

const main = require('./main'),
	{Backbone, common, etc, oneeSama, options, stackBlur, state} = main;

const BackgroundView = Backbone.View.extend({
	tagName: 'style',
	colourMap: {
		glass: {
			normal: 'rgba(40, 42, 46, 0.5)',
			editing: 'rgba(145, 145, 145, 0.5)'
		},
		ocean: {
			normal: 'rgba(28, 29, 34, 0.781)',
			editing: 'rgba(44, 57, 71, 0.88)'
		}
	},
	initialize() {
		document.head.append(this.el);
		document.body.append(etc.parseDOM('<div id="user_bg"></div>'));
		this.render();

		main.reply('background:store', this.store, this);
		this.listenTo(options, {
			'change:userBG': this.render,
			'change:illyaBGToggle': this.render,
			'change:illyaMuteToggle': this.render,
			'change:theme': this.render,
			'workModeTOG': this.render
		});
	},
	// Store image as dataURL in localStorage
	store(target) {
		// This could take a while, so loading animation
		main.request('loading:show');
		let reader = new FileReader();
		reader.readAsDataURL(target.files[0]);
		reader.onload = event => {
			let img = new Image();
			img.onload = () => {
				// Convert to JPEG
				let canvas = document.createElement("canvas");
				canvas.width = img.width;
				canvas.height = img.height;
				canvas
					.getContext('2d')
					.drawImage(img, 0, 0, img.width, img.height);
				localStorage.background = canvas.toDataURL('image/jpeg', 0.95);

				// Generate a blurred version of the background to use for
				// posts, modals, etc.
				stackBlur.canvas(canvas, 0, 0, img.width, img.height, 10);
				localStorage.blurred = canvas.toDataURL('image/jpeg', 0.95);

				main.request('loading:hide');

				// Apply new background
				if (options.get('userBG'))
					this.render();
			};
			img.src = event.target.result;
		};
	},
	render() {
		const {el} = this;
		el.innerHTML = '';
		if (options.get('illyaBGToggle') && state.hotConfig.get('ILLYA_DANCE'))
			this.renderIllya();
		else if (options.get('userBG') && !main.oneeSama.workMode)
			this.renderBackground();
	},
	renderBackground() {
		const bg = localStorage.background;
		if (!bg)
			return;
		let html = common.parseHTML
			`#user_bg {
				background: url(${bg}) no-repeat fixed center;
				background-size: cover;
			}`;

		// Add blurred background image to elements, if theme is glass or ocean
		const theme = main.oneeSama.workMode
			? state.hotConfig.get('DEFAULT_CSS') : options.get('theme')
		if (theme === 'glass' || theme === 'ocean') {
			const {blurred} = localStorage;
			if (blurred)
				html += ' ' + this.renderGlass(theme, blurred);
		}
		this.el.innerHTML = html;
	},
	renderGlass(theme, blurred) {
		const {normal, editing} = this.colourMap[theme];
		return common.parseHTML
			`.glass {
				background:
					linear-gradient(${normal}, ${normal}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}
			.glass.editing, .editing .background {
				background:
					linear-gradient(${editing}, ${editing}),
					url(${blurred}) center fixed no-repeat;
				background-size: cover;
			}
			.background {
				padding: 10px;
				margin: 2px;
			}`;
	},
	renderIllya() {
		const urlBase = main.config.MEDIA_URL + 'illya.';
		this.el.innerHTML = common.parseHTML
			`<video autoplay loop ${options.get('illyaMuteToggle') && 'muted'}>
				<source src="${urlBase + 'webm'}" type="video/webm">
				<source src="${urlBase + 'mp4'}" type="video/mp4">
			</video>`;
	}
});

main.defer(() => module.exports = new BackgroundView());
