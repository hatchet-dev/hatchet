import { ObjectFieldTemplateProps } from '@rjsf/utils';
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion';

import { DEFAULT_COLLAPSED } from '../json-form';

export const CollapsibleSection = (props: ObjectFieldTemplateProps) => {
  if (!props.title) {
    return props.properties.map((element, i) => (
      <div className="property-wrapper ml-4" key={i}>
        {element.content}
      </div>
    ));
  }

  return (
    <Accordion
      type="single"
      collapsible
      className="w-full"
      defaultValue={DEFAULT_COLLAPSED.includes(props.title) ? 'closed' : 'open'}
    >
      <AccordionItem value="open">
        <AccordionTrigger>{props.title}</AccordionTrigger>
        <AccordionContent>
          {props.description}
          {props.properties?.length > 0 ? (
            props.properties.map((element, i) => (
              <div className="property-wrapper ml-4" key={i}>
                {element.content}
              </div>
            ))
          ) : (
            <div className="ml-4">empty state</div>
          )}
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
};
