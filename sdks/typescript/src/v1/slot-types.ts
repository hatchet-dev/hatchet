// eslint-disable-next-line no-shadow
export enum SlotType {
  Default = 'default',
  Durable = 'durable',
}

export type SlotConfig = Partial<Record<SlotType, number>>;
