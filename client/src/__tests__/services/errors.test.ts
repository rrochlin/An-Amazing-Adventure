import { describe, it, expect } from 'vitest';
import { friendlyError } from '@/services/errors';

describe('friendlyError', () => {
   it('turns server codes into words, and leaves sentences alone', () => {
      expect(friendlyError('dialogue_choice_required')).toBe('Choose one of the options first.');
      expect(friendlyError('Mother told you not to go past the meadow.')).toBe('Mother told you not to go past the meadow.');
   });
});
