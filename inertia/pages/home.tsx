import { Head } from '@inertiajs/react'

import { Button } from '@kibamail/owly'

export default function Home() {
  return (
    <>
      <Head title="Homepage" />

      <div className="w-full h-16">
        <div className="flex max-w-6xl mx-auto h-full items-center justify-between">
          <p>Kibaship</p>

          <div className="flex gap-4">
            <Button variant="secondary">Login</Button>
            <Button variant="primary" asChild>
              <a href="/auth/redirect">Get started for free</a>
            </Button>
          </div>
        </div>
      </div>
    </>
  )
}
