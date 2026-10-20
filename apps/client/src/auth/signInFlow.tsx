import { createContext, use, useMemo, useState, type ReactNode } from 'react';

/**
 * Carries the email from the first sign-in step to the code step. It stays
 * in memory rather than in the URL so the address does not end up in
 * browser history; reloading the code step starts over.
 */
type SignInFlow = {
  email: string | null;
  setEmail: (email: string) => void;
};

const SignInFlowContext = createContext<SignInFlow | null>(null);

export function SignInFlowProvider({ children }: { children: ReactNode }) {
  const [email, setEmail] = useState<string | null>(null);
  const value = useMemo(() => ({ email, setEmail }), [email]);
  return <SignInFlowContext value={value}>{children}</SignInFlowContext>;
}

export function useSignInFlow(): SignInFlow {
  const ctx = use(SignInFlowContext);
  if (!ctx) throw new Error('useSignInFlow must be used inside <SignInFlowProvider>');
  return ctx;
}
