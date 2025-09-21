import { useState } from 'react'
import { Button } from '@kibamail/owly/button'
import { CreateApplicationDialog } from './CreateApplicationDialog'
import { PlusIcon } from '~/Components/Icons/plus.svg'

/**
 * Example component demonstrating how to use the CreateApplicationDialog
 */
export function CreateApplicationDialogExample() {
  const [isDialogOpen, setIsDialogOpen] = useState(false)

  return (
    <div className="p-8 space-y-8">
      <h2 className="text-xl font-semibold">Create Application Dialog Example</h2>
      
      <div className="space-y-4">
        <p className="text-owly-content-secondary">
          Click the button below to open the Create Application dialog.
        </p>
        
        <Button 
          variant="primary" 
          onClick={() => setIsDialogOpen(true)}
          className="flex items-center gap-2"
        >
          <PlusIcon className="w-4 h-4" />
          Create Application
        </Button>
      </div>

      <CreateApplicationDialog 
        isOpen={isDialogOpen}
        onOpenChange={setIsDialogOpen}
      />
    </div>
  )
}
