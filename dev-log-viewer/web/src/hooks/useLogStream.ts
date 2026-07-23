import { useEffect } from 'react'
import { buildStreamURL, decodeStreamEnvelope, type EventSourceFactory, type EventSourceLike } from '../api/stream'
import type { ConnectionStatus, StreamEnvelope } from '../types/api'

export interface LogStreamControllerOptions {
  createEventSource: EventSourceFactory
  onEnvelope(envelope: StreamEnvelope): void
  onMalformed(): void
  onStatus(status: ConnectionStatus): void
}

export class LogStreamController {
  private source: EventSourceLike | null = null

  constructor(private readonly options: LogStreamControllerOptions) {}

  connect(services: readonly string[], tail = 300): void {
    this.close()
    this.options.onStatus('connecting')
    const source = this.options.createEventSource(buildStreamURL(services, tail))
    this.source = source
    source.onopen = () => this.options.onStatus('connected')
    source.onerror = () => this.options.onStatus('reconnecting')
    for (const eventName of ['snapshot_start', 'log', 'snapshot_end', 'reset', 'dropped', 'heartbeat', 'status']) {
      source.addEventListener(eventName, (event) => {
        try {
          this.options.onEnvelope(decodeStreamEnvelope(event.data))
        } catch {
          this.options.onMalformed()
        }
      })
    }
  }

  close(): void {
    if (this.source) {
      this.source.close()
      this.source = null
      this.options.onStatus('disconnected')
    }
  }
}

export function useLogStream(
  controller: LogStreamController,
  services: readonly string[],
  tail: number,
): void {
  useEffect(() => {
    controller.connect(services, tail)
    return () => controller.close()
  }, [controller, services, tail])
}
