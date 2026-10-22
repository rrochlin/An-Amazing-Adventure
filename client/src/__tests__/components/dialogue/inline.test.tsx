// @vitest-environment jsdom
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { renderInline } from '@/components/dialogue/inline';

const html = (text: string) => render(<p>{renderInline(text)}</p>).container.innerHTML;

describe('renderInline', () => {
   it('styles *italic* and **bold**', () => {
      expect(html('a *soft* and **loud** word')).toBe(
         '<p>a <em>soft</em> and <strong>loud</strong> word</p>',
      );
   });

   it('styles text that is still typing in, without a stray marker', () => {
      expect(html('It is *gon')).toBe('<p>It is <em>gon</em></p>');
   });

   it('leaves plain text and lone asterisks alone', () => {
      expect(renderInline('No markers here.')).toBe('No markers here.');
      expect(html('two * three')).toBe('<p>two * three</p>');
   });
});
