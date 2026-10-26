/**
 * What the player reads for a server error code (the panel saw the raw
 * "dialogue_choice_required"). Anything else the server sends is already a
 * sentence (a blocked place's reason) and is shown as it is.
 */
const ERROR_TEXT: Record<string, string> = {
   dialogue_choice_required: 'Choose one of the options first.',
   in_conversation: "You're in a conversation: press Esc or Leave to walk away.",
   not_here: "They aren't here any more.",
   internal_error: 'Something went wrong on our side. Try again in a moment.',
   campaign_not_found: "This adventure's story couldn't be loaded.",
   user_not_found: 'Your account could not be found. Try signing in again.',
};

export function friendlyError(code: string): string {
   return ERROR_TEXT[code] ?? code;
}
