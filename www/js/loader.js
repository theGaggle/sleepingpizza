/*
Selects and loads the client files
 */

(function () {
	// Check for browser compatibility by trying to detect some ES6 features
	function check(func) {
		try {
			return eval('(function(){' + func + '})();');
		}
		catch(e) {
			return false;
		}
	}

	var tests = [
		// Constants
		'"use strict"; const foo = 123; return foo === 123;',
		// Block scoping
		'"use strict";  const bar = 123; {const bar = 456;} return bar===123;',
		// Template strings
		'var a = "ba"; return `foo bar${a + "z"}` === "foo barbaz";',
		// for...of
		'var arr = [5]; for (var item of arr) return item === 5;'
	];
	var legacy;
	for (var i = 0; i < tests.length; i++) {
		if (!check(tests[i])) {
			// Load client with full ES5 compliance
			legacy = true;
			break;
		}
	}

	var $script = require('scriptjs'),
		base = config.MEDIA_URL + 'js/',
		end = '.js?v=' + clientHash;
	$script(base + 'lang/' + lang + end, function() {
		var client = legacy ? 'legacy' : 'client';
		$script(base + client + end, function () {
			if (typeof IDENT !== 'undefined') {
				$script('../mod.js', function () {
					require('mod');
				});
			}
		});
	});
})();
