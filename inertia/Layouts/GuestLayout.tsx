import { AgreeToTermsAndPolicy } from '~/Components/agree-to-terms-and-policy';
import { Text } from '@kibamail/owly/text';
import type React from 'react';
import { CheckCircleSolidIcon } from '~/Components/Icons/check-circle-solid.svg';
import { usePage } from '@inertiajs/react';

type AuthLayoutProps = { children?: React.ReactNode };

function PasswordResetsFlowLayout({
  children,
}: React.PropsWithChildren<AuthLayoutProps>) {
  return (
    <div className="flex h-screen w-full flex-col overflow-y-hidden">
      <div className="w-full px-4 py-8 md:px-10">
        <img src="/logos/full-light.svg" className="h-8" alt="Kibamail Logo" />
      </div>

      <div className="grow">{children}</div>

      <AgreeToTermsAndPolicy />
    </div>
  );
}

export default function GuestLayout({
  children,
}: React.PropsWithChildren<AuthLayoutProps>) {
  const {url} = usePage();

  if (
    url?.includes('password') ||
    url?.includes('verification')
  ) {
    return <PasswordResetsFlowLayout>{children}</PasswordResetsFlowLayout>;
  }

  return (
    <div className="grid h-screen w-full grid-cols-1 gap-0 overflow-y-auto lg:grid-cols-2">
      <div className="kb-background-secondary flex h-full w-full flex-col">
        <div className="w-full px-4 py-8 md:px-10">
          <a href="/">
            <img
              src="/logos/full-light.svg"
              className="h-8"
              alt="Kibamail Logo"
            />
          </a>
        </div>
        <div className="grow px-5 lg:px-0">{children}</div>

        <AgreeToTermsAndPolicy />
      </div>

      <div className="kb-background-brand-hover relative hidden h-full w-full flex-col pt-48 lg:flex">
        <div className="flex h-12 w-full max-w-lg flex-col pl-24">
          <ProductFeatureGrid />
        </div>

        <div className="absolute bottom-48 h-px w-full border-t border-white/10" />
        <div className="absolute bottom-36 h-10 w-full border-t border-b border-white/10 bg-black/5" />
      </div>
    </div>
  );
}

function ProductFeatureGrid() {
  return (
    <div className="grid grid-cols-1 gap-8">
      <ProductFeature
        title="No subscriptions, 6,573 free emails per month"
        description="No subscriptions plans, no payment contracts, and your free email sends reset every
      month, forever."
      />
      <ProductFeature
        title="Unlimited tracking, contacts and automations."
        description="No limits on usage. Only pay for the successful emails you send via Kibamail."
      />
      <ProductFeature
        title="Open source, transparent and community driven"
        description="Audit our code and business yourself. No secrets, no tricks, no hidden fees."
      />
    </div>
  );
}

interface ProductFeatureProps {
  title?: string;
  description?: string;
}

function ProductFeature({ title, description }: ProductFeatureProps) {
  return (
    <div className="flex items-start gap-x-2">
      <CheckCircleSolidIcon className="kb-content-notice shrink-0" />

      <div className="flex grow flex-col gap-y-2">
        <Text className="kb-content-primary-inverse" size="lg">
          {title}
        </Text>

        <Text className="kb-content-tertiary-inverse">{description}</Text>
      </div>
    </div>
  );
}
