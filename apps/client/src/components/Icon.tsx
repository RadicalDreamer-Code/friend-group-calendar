import Ionicons from '@expo/vector-icons/Ionicons';
import type { ComponentProps } from 'react';

export type IconName = ComponentProps<typeof Ionicons>['name'];

type IconProps = {
  name: IconName;
  size?: number;
  color: string;
};

/** Thin wrapper so the icon set can be swapped in one place. */
export function Icon({ name, size = 22, color }: IconProps) {
  return <Ionicons name={name} size={size} color={color} aria-hidden />;
}
