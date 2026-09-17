import { type AgentUseCaseKey } from './prompt-templates';

// A use case, or the sentinel for the freeform "describe your own" choice.
// Shared by the stepper and the use-case graphics so the graphics module does
// not import from onboarding-steps (which would be circular).
export type UseCaseChoice = AgentUseCaseKey | 'custom';
