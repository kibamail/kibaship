import { useEffect, useState } from 'react'

/**
 * A custom hook that manages state based on a URL query parameter
 * and listens for changes to that parameter in the browser URL
 * 
 * @param parameterName - The name of the query parameter to track
 * @param defaultValue - Default value when parameter is not present
 * @returns [value, setValue] - Current parameter value and setter function
 */
export function useQueryParameterState<T extends string | null = string | null>(
  parameterName: string,
  defaultValue: T = null as T
): T {
  const getParameterValue = (): T => {
    if (typeof window === 'undefined') return defaultValue

    const url = new URL(window.location.href)
    const paramValue = url.searchParams.get(parameterName)

    return (paramValue ?? defaultValue) as T
  }

  const [value, setValue] = useState<T>(getParameterValue)

  useEffect(() => {
    const handleUrlChange = () => {
      const newValue = getParameterValue()
      setValue(newValue)
    }

    window.addEventListener('popstate', handleUrlChange)

    const originalPushState = window.history.pushState
    const originalReplaceState = window.history.replaceState

    window.history.pushState = function (...args) {
      originalPushState.apply(window.history, args)
      handleUrlChange()
    }

    window.history.replaceState = function (...args) {
      originalReplaceState.apply(window.history, args)
      handleUrlChange()
    }

    handleUrlChange()

    return () => {
      window.removeEventListener('popstate', handleUrlChange)
      window.history.pushState = originalPushState
      window.history.replaceState = originalReplaceState
    }
  }, [parameterName, defaultValue])

  return value
}
