import type { EntryCover } from '@/api/client';

/** What each cover scene is called, e.g. for the picker and screen readers. */
export const coverLabels = {
  lake: 'Lake',
  mountains: 'Mountains',
  beach: 'Beach',
  city: 'City',
  night_out: 'Night out',
  dinner: 'Dinner',
  game_night: 'Game night',
} as const satisfies Record<EntryCover, string>;

/** Every cover scene, in the order the picker shows them. */
export const coverScenes = Object.keys(coverLabels) as EntryCover[];
