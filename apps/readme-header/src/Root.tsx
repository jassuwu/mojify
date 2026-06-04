import {Composition} from 'remotion';
import {HeaderDemo} from './HeaderDemo';
import {SourceWord} from './SourceWord';

export const RemotionRoot: React.FC = () => {
  return (
    <>
      {/* The final animated header. Imports the generated character image, so it must
          be rendered AFTER the pipeline produces public/charword.png. */}
      <Composition
        id="ReadmeHeader"
        component={HeaderDemo}
        durationInFrames={48}
        fps={12}
        width={1440}
        height={480}
      />
      {/* Helper still: the clean wordmark that Mojify converts into the reveal image. */}
      <Composition
        id="ReadmeHeaderSource"
        component={SourceWord}
        durationInFrames={1}
        fps={12}
        width={1440}
        height={480}
      />
    </>
  );
};
