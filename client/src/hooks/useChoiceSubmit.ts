import { useCallback, useEffect, useRef, useState } from 'react';

/**
 * Sends one dialogue choice at a time. The choice buttons stay disabled
 * until the server answers: any new campaign state after the choice, or an
 * error. Waiting for "no choice pending" isn't enough, because a scene can
 * open straight into its own choice (Briarwell's gift stall → birthday).
 */
export function useChoiceSubmit<S>(
   campaignState: S,
   error: unknown,
   send: (choiceId: number) => void,
) {
   const [submitting, setSubmitting] = useState(false);
   const lock = useRef(false);
   const sentFrom = useRef<S>(campaignState);

   const release = useCallback(() => {
      lock.current = false;
      setSubmitting(false);
   }, []);

   const submit = useCallback(
      (choiceId: number) => {
         if (lock.current) return;
         lock.current = true;
         sentFrom.current = campaignState;
         setSubmitting(true);
         send(choiceId);
      },
      [campaignState, send],
   );

   useEffect(() => {
      if (!submitting) return;
      if (error || campaignState !== sentFrom.current) release();
   }, [campaignState, error, submitting, release]);

   return { submitting, submit, release };
}
