import {Composition} from 'remotion';
import {HeaderDemo} from './HeaderDemo';

export const RemotionRoot: React.FC = () => {
  return (
    <Composition
      id="ReadmeHeader"
      component={HeaderDemo}
      durationInFrames={48}
      fps={12}
      width={960}
      height={320}
    />
  );
};
