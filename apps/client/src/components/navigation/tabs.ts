import type { Href } from 'expo-router';

import type { IconName } from '../Icon';

export type AppTab = {
  name: string;
  href: Href;
  label: string;
  icon: IconName;
  iconFocused: IconName;
};

export const appTabs: AppTab[] = [
  { name: 'calendars', href: '/', label: 'Calendars', icon: 'calendar-outline', iconFocused: 'calendar' },
  { name: 'friends', href: '/friends', label: 'Friends', icon: 'people-outline', iconFocused: 'people' },
  { name: 'profile', href: '/profile', label: 'Profile', icon: 'person-circle-outline', iconFocused: 'person-circle' },
];
