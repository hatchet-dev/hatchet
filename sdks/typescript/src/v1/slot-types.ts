export enum SlotType {
  Default = 'default',
  Durable = 'durable',
}

export type SlotCapacities = Partial<Record<SlotType, number>>;
