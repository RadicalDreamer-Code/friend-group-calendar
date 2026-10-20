import { Redirect, router } from 'expo-router';
import { useEffect, useState } from 'react';
import { View } from 'react-native';

import { useRequestCode, useVerifyCode } from '@/api/account';
import { ApiError, errorMessage } from '@/api/client';
import { useSignInFlow } from '@/auth/signInFlow';
import { AuthScreen } from '@/components/AuthScreen';
import { Button } from '@/components/Button';
import { CodeInput } from '@/components/CodeInput';
import { Text } from '@/components/Text';
import { createStyles } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';

const CODE_LENGTH = 6;
/** Matches the API's per-address cooldown between codes. */
const RESEND_SECONDS = 30;

export default function VerifyScreen() {
  const { email } = useSignInFlow();
  if (!email) return <Redirect href="/sign-in" />;
  return <VerifyForm email={email} />;
}

function VerifyForm({ email }: { email: string }) {
  const styles = useStyles();
  const [code, setCode] = useState('');
  const verify = useVerifyCode();
  const resend = useRequestCode();
  const resendIn = useCountdown(RESEND_SECONDS, resend.submittedAt);

  // Submit as soon as the last digit is in, including after paste or autofill.
  useEffect(() => {
    if (code.length === CODE_LENGTH && verify.isIdle) {
      verify.mutate({ email, code }, { onError: () => setCode('') });
    }
  }, [code, email, verify]);

  const expired = verify.error instanceof ApiError && verify.error.code === 'code_expired';
  const message =
    resend.isSuccess && verify.isIdle
      ? 'We sent a new code.'
      : verify.error
        ? errorMessage(verify.error)
        : resend.error
          ? errorMessage(resend.error)
          : null;

  return (
    <AuthScreen
      title="Check your email"
      subtitle={
        <>
          We sent a 6-digit code to{' '}
          <Text variant="callout" style={styles.email}>
            {email}
          </Text>
          . It expires in 10 minutes.
        </>
      }
    >
      <CodeInput
        value={code}
        onChange={(next) => {
          setCode(next);
          if (!verify.isIdle) verify.reset();
        }}
        error={verify.isError}
        editable={!verify.isPending}
        autoFocus
      />
      <View style={styles.status}>
        {message ? (
          <Text variant="footnote" tone={verify.isError || resend.isError ? 'danger' : 'secondary'} aria-live="polite">
            {message}
          </Text>
        ) : verify.isPending ? (
          <Text variant="footnote" tone="secondary">
            Signing in…
          </Text>
        ) : null}
      </View>
      <View style={styles.actions}>
        <Button
          variant={expired ? 'primary' : 'plain'}
          title={resendIn > 0 ? `Send a new code in ${resendIn}s` : 'Send a new code'}
          disabled={resendIn > 0}
          loading={resend.isPending}
          onPress={() => {
            verify.reset();
            setCode('');
            resend.mutate(email);
          }}
        />
        <Button variant="plain" title="Use a different email" onPress={() => router.back()} />
      </View>
    </AuthScreen>
  );
}

/** Seconds left until `seconds` have passed since `from` (counts from mount when unset). */
function useCountdown(seconds: number, from: number | undefined): number {
  const [start] = useState(() => Date.now());
  const [now, setNow] = useState(() => Date.now());
  const since = from || start;
  const left = Math.max(0, Math.ceil((since + seconds * 1000 - now) / 1000));
  useEffect(() => {
    if (left === 0) return;
    const id = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(id);
  }, [left]);
  return left;
}

const useStyles = createStyles((c) => ({
  email: {
    color: c.text,
    fontWeight: '600',
  },
  status: {
    minHeight: 18,
    alignItems: 'center',
  },
  actions: {
    alignItems: 'center',
    gap: space.xs,
  },
}));
