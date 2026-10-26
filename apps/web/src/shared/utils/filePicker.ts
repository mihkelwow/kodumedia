import { ApiError, api } from '../../api/client'
import type { FilePickerRequest } from '../../api/types'

export type NativePickerOutcome =
  | { kind: 'selected'; path: string }
  | { kind: 'cancelled' }
  | { kind: 'unavailable' }

export async function openNativeFilePicker(input: FilePickerRequest): Promise<NativePickerOutcome> {
  try {
    const result = await api.filesystem.pick(input)
    if (!result?.path) return { kind: 'cancelled' }
    return { kind: 'selected', path: result.path }
  } catch (caught) {
    if (caught instanceof ApiError && (caught.status === 503 || caught.status === 404)) {
      return { kind: 'unavailable' }
    }
    throw caught
  }
}
