import { createContext, Dispatch, SetStateAction, useContext } from 'react'
import { Application } from '~/types'

export const ProjectPageContext = createContext<{
  selectedApplication: Application | null
  setSelectedApplication: Dispatch<SetStateAction<Application | null>>
  applicationsMap: Record<string, Application>
}>({
  selectedApplication: null,
  setSelectedApplication() {},
  applicationsMap: {},
})

export function useProjectPageContext() {
  return useContext(ProjectPageContext)
}
