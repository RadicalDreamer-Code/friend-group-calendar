import { Children, Fragment, type ReactNode } from 'react';
import { Pressable, View } from 'react-native';

import { createStyles, useTheme } from '@/theme/ThemeProvider';
import { hairline, radius, space } from '@/theme/tokens';

import { Icon } from './Icon';
import { Text } from './Text';

type ListSectionProps = {
  title?: string;
  footer?: string;
  /** Left inset of the separators; lines them up with text after a leading avatar. */
  separatorInset?: number;
  children: ReactNode;
};

/** An inset grouped list (like iOS Settings) with hairline separators between rows. */
export function ListSection({ title, footer, separatorInset, children }: ListSectionProps) {
  const styles = useStyles();
  const rows = Children.toArray(children);
  return (
    <View style={styles.root}>
      {title ? (
        <Text variant="footnote" tone="secondary" style={styles.title}>
          {title}
        </Text>
      ) : null}
      <View style={styles.group}>
        {rows.map((row, i) => (
          <Fragment key={i}>
            {i > 0 ? (
              <View style={[styles.separator, separatorInset !== undefined && { marginLeft: separatorInset }]} />
            ) : null}
            {row}
          </Fragment>
        ))}
      </View>
      {footer ? (
        <Text variant="footnote" tone="tertiary" style={styles.footer}>
          {footer}
        </Text>
      ) : null}
    </View>
  );
}

type ListRowProps = {
  /** Shown before the label, e.g. an avatar. */
  leading?: ReactNode;
  label?: string;
  /** Smaller second line under the label. */
  sublabel?: string;
  /** Right-aligned content: a value string or any element. */
  detail?: ReactNode;
  children?: ReactNode;
  /** Makes the row a button. */
  onPress?: () => void;
  /** Shows a disclosure chevron, for rows that open another screen. */
  chevron?: boolean;
  /** Accent (or red) label, for rows that are actions rather than values. */
  action?: 'default' | 'destructive';
  disabled?: boolean;
};

export function ListRow({
  leading,
  label,
  sublabel,
  detail,
  children,
  onPress,
  chevron,
  action,
  disabled,
}: ListRowProps) {
  const styles = useStyles();
  const { colors } = useTheme();
  const tone = action === 'destructive' ? 'danger' : action === 'default' ? 'accent' : 'primary';

  const content = (
    <>
      {leading}
      {label ? (
        <View style={styles.labels}>
          <Text tone={tone} numberOfLines={1}>
            {label}
          </Text>
          {sublabel ? (
            <Text variant="footnote" tone="secondary" numberOfLines={1}>
              {sublabel}
            </Text>
          ) : null}
        </View>
      ) : null}
      {typeof detail === 'string' ? (
        <Text tone="secondary" numberOfLines={1} style={styles.detail}>
          {detail}
        </Text>
      ) : (
        detail
      )}
      {children}
      {chevron ? <Icon name="chevron-forward" size={18} color={colors.textTertiary} /> : null}
    </>
  );

  if (!onPress) return <View style={styles.row}>{content}</View>;
  return (
    <Pressable
      role="button"
      onPress={onPress}
      disabled={disabled}
      aria-disabled={disabled}
      style={({ pressed, hovered }: { pressed: boolean; hovered?: boolean }) => [
        styles.row,
        (pressed || hovered) && styles.rowPressed,
        disabled && styles.rowDisabled,
      ]}
    >
      {content}
    </Pressable>
  );
}

const useStyles = createStyles((c) => ({
  root: {
    marginTop: space.xl,
  },
  title: {
    paddingHorizontal: space.lg,
    marginBottom: space.sm,
  },
  group: {
    backgroundColor: c.surface,
    borderRadius: radius.md,
    borderWidth: hairline,
    borderColor: c.hairline,
    overflow: 'hidden',
  },
  separator: {
    height: hairline,
    backgroundColor: c.hairline,
    marginLeft: space.lg,
  },
  footer: {
    paddingHorizontal: space.lg,
    marginTop: space.sm,
  },
  row: {
    minHeight: 48,
    paddingHorizontal: space.lg,
    paddingVertical: space.md,
    flexDirection: 'row',
    alignItems: 'center',
    gap: space.md,
  },
  rowPressed: {
    backgroundColor: c.fill,
  },
  rowDisabled: {
    opacity: 0.4,
  },
  labels: {
    flex: 1,
    gap: space.xxs,
  },
  detail: {
    flexShrink: 1,
  },
}));
