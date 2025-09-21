import { WorldReady } from "../services/api.game";

const MAX_WAIT_TIME = 60_000; // 1 minute in milliseconds
const INITIAL_BACKOFF = 1_000; // Start with 1 second
const MAX_BACKOFF = 8_000; // Maximum backoff of 8 seconds

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));

export const pollWorldStatus = async (uuid: string) => {
  const startTime = Date.now();
  let backoff = INITIAL_BACKOFF;
  let attempts = 0;

  while (Date.now() - startTime < MAX_WAIT_TIME) {
    attempts++;

    const response = await WorldReady(uuid);
    if (response.ready) {
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
