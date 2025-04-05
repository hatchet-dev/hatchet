import * as React from 'react';
import { Button, ButtonProps } from './button';
import { BookOpenIcon } from 'lucide-react';
import { Link } from 'react-router-dom';

type DocRef = {
  title: string;
  href: string;
};

interface DocsButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  prefix?: string;
  doc: DocRef;
  size?: ButtonProps['size'];
}

const baseDocsUrl = 'https://docs.hatchet.run';

export function DocsButton({
  doc,
  prefix = 'Learn more about ',
  size = 'sm',
  ...props
}: DocsButtonProps) {
  return (
    <Link to={`${baseDocsUrl}${doc.href}`} target="_blank">
      <Button variant="outline" {...props} size={size}>
        <BookOpenIcon className="w-4 h-4 mr-2" />
        {prefix} {doc.title}
      </Button>
    </Link>
  );
}
