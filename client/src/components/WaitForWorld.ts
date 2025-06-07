import axios from 'axios';

const APP_URI = import.meta.env.VITE_APP_URI || 'http://localhost:8080/';
const MAX_WAIT_TIME = 60000; // 1 minute in milliseconds
const INITIAL_BACKOFF = 1000; // Start with 1 second
const MAX_BACKOFF = 8000; // Maximum backoff of 8 seconds

const checkWorldReady = async () => {
	try {
		const response = await axios.get(`${APP_URI}worldready`);
		if (response.data.ready) {
			return true;
		}
		return false;
	} catch (err) {
		console.error('Error checking world status:', err);
		return false;
	}
};

const sleep = (ms: number) => new Promise(resolve => setTimeout(resolve, ms));

export const pollWorldStatus = async () => {
	const startTime = Date.now();
	let backoff = INITIAL_BACKOFF;
	let attempts = 0;

	while (Date.now() - startTime < MAX_WAIT_TIME) {
		attempts++;

		const isReady = await checkWorldReady();
		if (isReady) {
			return true;
		}

		// Calculate next backoff time
		const nextBackoff = Math.min(backoff * 1.5, MAX_BACKOFF);
		const remainingTime = MAX_WAIT_TIME - (Date.now() - startTime);
		const actualBackoff = Math.min(nextBackoff, remainingTime);

		if (actualBackoff <= 0) {
			break;
		}

		await sleep(actualBackoff);
		backoff = nextBackoff;
	}

	return false;
};
