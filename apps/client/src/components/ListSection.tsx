import { Children, Fragment, type ReactNode } from 'react';
import { View } from 'react-native';

import { createStyles } from '@/theme/ThemeProvider';
import { hairline, radius, space } from '@/theme/tokens';

import { Text } from './Text';

type ListSectionProps = {
  title?: string;
  footer?: string;
  children: ReactNode;
};

/** An inset grouped list (like iOS Settings) with hairline separators between rows. */
export function ListSection({ title, footer, children }: ListSectionProps) {
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
            {i > 0 ? <View style={styles.separator} /> : null}
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
  label?: string;
  /** Right-aligned content: a value string or any element. */
  detail?: ReactNode;
  children?: ReactNode;
};

export function ListRow({ label, detail, children }: ListRowProps) {
  const styles = useStyles();
  return (
    <View style={styles.row}>
      {label ? <Text style={styles.label}>{label}</Text> : null}
      {typeof detail === 'string' ? <Text tone="secondary">{detail}</Text> : detail}
      {children}
    </View>
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
  label: {
    flex: 1,
  },
}));
