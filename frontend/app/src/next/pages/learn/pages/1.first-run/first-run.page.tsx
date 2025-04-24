import { lessonPlan } from './first-run.lesson-plan';
import { Lesson } from '../../../../learn/components/lesson';

export default function FirstRunPage() {
  return <Lesson lesson={lessonPlan} />;
}
