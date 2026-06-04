import {AbsoluteFill} from 'remotion';
import {FIELD, Wordmark} from './Wordmark';

/**
 * `ReadmeHeaderSource` — the clean wordmark on a flat near-black field, nothing else.
 * The render pipeline renders one frame of this, runs it through real `mojify export`
 * to produce the character version, and feeds that image back into `ReadmeHeader` as
 * the reveal layer. Keeping it isolated guarantees Mojify converts ONLY the word
 * (no command line, no chrome), so the character output is clean and legible.
 */
export const SourceWord: React.FC = () => (
  <AbsoluteFill style={{backgroundColor: FIELD}}>
    <Wordmark />
  </AbsoluteFill>
);
