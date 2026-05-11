'use strict';

/**
 * Ensures crypto.getRandomValues exists on the builtin `crypto` namespace.
 * Vite resolves `import crypto from "node:crypto"` and expects Web Crypto parity.
 */
const crypto = require('node:crypto');
if (typeof crypto.getRandomValues !== 'function') {
	if (crypto.webcrypto?.getRandomValues) {
		crypto.getRandomValues = function getRandomValues(typedArray) {
			return crypto.webcrypto.getRandomValues(typedArray);
		};
	} else if (typeof crypto.randomFillSync === 'function') {
		crypto.getRandomValues = function getRandomValues(typedArray) {
			crypto.randomFillSync(typedArray);
			return typedArray;
		};
	}
}
