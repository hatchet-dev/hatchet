import { Button } from '../ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';

export type DocPage = {
  title: string;
  href: string;
};

export const DocsButton = ({ doc }: { doc: DocPage }) => {
  const { open } = useSidePanel();

  const handleClick = () => {
    open({
      type: 'docs',
      content: doc,
    });
  };

  return (
    <Button onClick={handleClick}>
      Learn more about {doc.title.toLocaleLowerCase()}
    </Button>
  );
};
