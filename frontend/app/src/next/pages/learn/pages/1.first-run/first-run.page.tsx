import { lessonPlan } from './first-run.lesson-plan';
import { Lesson } from '../../components/lesson';

export default function FirstRunPage() {
  return <Lesson lesson={lessonPlan} />;
}
