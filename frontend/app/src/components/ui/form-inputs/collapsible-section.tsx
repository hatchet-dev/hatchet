import { ObjectFieldTemplateProps } from '@rjsf/utils';
import { useState } from 'react';
import { DEFAULT_COLLAPSED } from '../json-form';

export const CollapsibleSection = (props: ObjectFieldTemplateProps) => {
  const [open, setOpen] = useState(!DEFAULT_COLLAPSED.includes(props.title));

  return (
    <div>
      {props.title && (
        <div
          onClick={() => setOpen((x) => !x)}
          className="border-b-2 mb-2 border-gray-500 pb-2 text-xl font-bold flex items-center cursor-pointer"
        >
          <svg
            className={`mr-2 h-6 w-6 ${open ? 'rotate-180' : ''}`}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <path d="M19 9l-7 7-7-7" />
          </svg>

          {props.title}
        </div>
      )}
      {props.description}
      {open &&
        (props.properties.length > 0 ? (
          props.properties.map((element, i) => (
            <div className="property-wrapper ml-4" key={i}>
              {element.content}
            </div>
          ))
        ) : (
          <div className="ml-4">empty state</div>
        ))}
    </div>
  );
};
