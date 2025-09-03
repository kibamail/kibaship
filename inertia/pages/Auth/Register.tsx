import { Head, Link, useForm } from '@inertiajs/react';
import { FormEventHandler } from 'react';

import {
  AuthMethodsDivider,
  Oauth2Methods,
  PageContainer,
  PageTitle,
} from '~/Components/auth';
import { PasswordField } from '~/Components/password-field';
import GuestLayout from '~/Layouts/GuestLayout';
import { Button } from '@kibamail/owly/button';
import { Text } from '@kibamail/owly/text';
import * as TextField from '@kibamail/owly/text-field';
import { FlashMessage } from '~/Components/flash_message';

interface RegisterPageProps {}

export default function Register({}: RegisterPageProps) {
  const { data, setData, post, processing, errors, reset } = useForm({
    email: '',
    password: '',
  });

  const submit: FormEventHandler = (e) => {
    e.preventDefault();

    post('/auth/register', {
      onFinish: () => reset('password'),
    });
  };

  return (
    <GuestLayout>
      <Head title="Register" />
      <PageContainer>
        <PageTitle
          title={'Welcome to a new world of Emailing.'}
          description={
            'Choose your preferred method to access powerful emailing tools.'
          }
        />

        <FlashMessage className="mt-10" />

        <Oauth2Methods page="register" />

        <AuthMethodsDivider>Or signup with</AuthMethodsDivider>

        <form className="flex w-full flex-col py-4" onSubmit={submit}>
          <div className="grid grid-cols-1 gap-4">
            <TextField.Root
              id="email"
              name="email"
              required
              type="email"
              placeholder="Enter your work email address"
              value={data.email}
              onChange={(e) => setData('email', e.target.value)}
            >
              <TextField.Label htmlFor="email">Email address</TextField.Label>
              {errors?.email ? (
                <TextField.Error>{errors?.email}</TextField.Error>
              ) : null}
            </TextField.Root>

            <div className="relative">
              <PasswordField
                required
                strengthIndicator
                name="password"
                id="new-password"
                placeholder="Choose a password"
                value={data.password}
                onChange={(e) => setData('password', e.target.value)}
              >
                <TextField.Label htmlFor="password">Password</TextField.Label>

                {errors?.password ? (
                  <TextField.Error className="mt-6">
                    {errors?.password}
                  </TextField.Error>
                ) : null}
              </PasswordField>
            </div>
          </div>

          <Button
            type="submit"
            loading={processing}
            width="full"
            className="mt-6"
          >
            Sign up
          </Button>
        </form>

        <div className="flex justify-center">
          <Text>
            Already have an account?
            <Link className="kb-content-info ml-2 underline" href={'/login'}>
              Login
            </Link>
          </Text>
        </div>
      </PageContainer>
    </GuestLayout>
  );
}
