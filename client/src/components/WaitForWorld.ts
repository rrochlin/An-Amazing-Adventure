import { WorldReady } from "../services/api.game";

const MAX_WAIT_TIME = 180_000; // 3 minutes — world-gen can take 60-90s
const INITIAL_BACKOFF = 5_000; // Start with 5 seconds
const MAX_BACKOFF = 15_000; // Cap at 15s so we stay responsive

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
