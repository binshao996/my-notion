import * as Y from "yjs";
import * as encoding from "lib0/encoding";
import * as decoding from "lib0/decoding";

// Simple awareness store (lightweight alternative to y-protocols/awareness)
export type AwarenessState = {
  name: string;
  color: string;
  cursor?: { blockId: string; from: number; to: number } | null;
};

type AwarenessChangeHandler = (states: Map<number, AwarenessState>, origin: number | null) => void;

// Event emitter types for yCursorPlugin compatibility
type EventHandler = (...args: any[]) => void;

export class Awareness {
  clientID: number;
  private states: Map<number, AwarenessState> = new Map();
  private handlers: AwarenessChangeHandler[] = [];
  private eventHandlers: Map<string, Set<EventHandler>> = new Map();

  constructor() {
    this.clientID = Math.floor(Math.random() * 0x7fffffff);
  }

  getStates(): Map<number, AwarenessState> {
    return this.states;
  }

  getLocalState(): AwarenessState | null {
    return this.states.get(this.clientID) ?? null;
  }

  setLocalState(state: AwarenessState): void {
    this.states.set(this.clientID, state);
    this.fire({ added: [this.clientID], updated: [this.clientID], removed: [] }, this.clientID);
  }

  setLocalStateField(field: string, value: any): void {
    const current = this.getLocalState();
    if (!current) return;
    this.setLocalState({ ...current, [field]: value });
  }

  applyRemote(raw: { clientId: number; state: AwarenessState }[]): void {
    for (const { clientId, state } of raw) {
      this.states.set(clientId, state);
    }
    const changed = raw.map((s) => s.clientId);
    this.fire({ added: changed, updated: [], removed: [] }, null);
  }

  removeRemote(clientIds: number[]): void {
    for (const id of clientIds) {
      this.states.delete(id);
    }
    this.fire({ added: [], updated: [], removed: clientIds }, null);
  }

  // EventEmitter-compatible API (for yCursorPlugin)
  on(event: string, handler: EventHandler): void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(handler);
  }

  off(event: string, handler: EventHandler): void {
    this.eventHandlers.get(event)?.delete(handler);
  }

  private emit(event: string, ...args: any[]): void {
    this.eventHandlers.get(event)?.forEach((h) => h(...args));
  }

  // Legacy onChange/offChange (used by our own code)
  onChange(handler: AwarenessChangeHandler): void {
    this.handlers.push(handler);
  }

  offChange(handler: AwarenessChangeHandler): void {
    this.handlers = this.handlers.filter((h) => h !== handler);
  }

  private fire(_change: { added: number[]; updated: number[]; removed: number[] }, origin: number | null): void {
    for (const h of this.handlers) {
      h(this.states, origin);
    }
    this.emit("change", [{ added: _change.added, updated: _change.updated, removed: _change.removed }], origin);
  }

  destroy(): void {
    this.states.clear();
    this.handlers = [];
    this.eventHandlers.clear();
  }
}

// Generated colors for cursor distinction
const USER_COLORS = [
  "#f87171", "#60a5fa", "#34d399", "#fbbf24",
  "#a78bfa", "#fb923c", "#4ade80", "#f472b6",
];

export class PageAwareness {
  ydoc: Y.Doc;
  awareness: Awareness;
  private ws: WebSocket | null = null;
  private pageId: number | null = null;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private destroyed = false;

  // Callbacks
  onConnected?: () => void;
  onDisconnected?: () => void;

  constructor() {
    this.ydoc = new Y.Doc();
    this.awareness = new Awareness();
  }

  connect(pageId: number, userName: string): void {
    this.pageId = pageId;
    this.destroyed = false;

    // Set initial awareness state
    const color = USER_COLORS[(pageId * 7 + this.awareness.clientID) % USER_COLORS.length];
    this.awareness.setLocalState({ name: userName, color, cursor: null });

    this.doConnect(pageId, userName);

    // Listen for Yjs updates
    this.ydoc.on("update", (update: Uint8Array) => {
      if (this.ws && this.ws.readyState === WebSocket.OPEN) {
        const encoder = encoding.createEncoder();
        encoding.writeUint8(encoder, 1); // sync_update
        encoding.writeUint8Array(encoder, update);
        this.ws.send(encoding.toUint8Array(encoder));
      }
    });

    // Listen for awareness changes
    this.awareness.onChange(() => {
      this.sendAwareness();
    });
  }

  private doConnect(pageId: number, _userName: string): void {
    const token = localStorage.getItem("token");
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const url = `${protocol}//${host}/ws/page/${pageId}?token=${token}`;

    const ws = new WebSocket(url);
    ws.binaryType = "arraybuffer";
    this.ws = ws;

    ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.onConnected?.();
      this.sendAwareness();
    };

    ws.onmessage = (event) => {
      if (event.data instanceof ArrayBuffer) {
        this.handleBinary(new Uint8Array(event.data));
      } else if (typeof event.data === "string") {
        this.handleJSON(event.data);
      }
    };

    ws.onclose = () => {
      this.onDisconnected?.();
      this.scheduleReconnect();
    };

    ws.onerror = () => {
      ws.close();
    };
  }

  private handleBinary(data: Uint8Array): void {
    if (data.length === 0) return;
    const decoder = decoding.createDecoder(data);
    const msgType = decoding.readUint8(decoder);

    switch (msgType) {
      case 0: { // sync_init — full document
        while (decoding.hasContent(decoder)) {
          const len = decoding.readVarUint(decoder);
          const update = decoding.readUint8Array(decoder, len);
          Y.applyUpdate(this.ydoc, update, "remote");
        }
        break;
      }
      case 1: { // sync_update — single update
        Y.applyUpdate(this.ydoc, new Uint8Array(data.buffer, data.byteOffset + 1, data.length - 1), "remote");
        break;
      }
    }
  }

  private handleJSON(data: string): void {
    try {
      const msg = JSON.parse(data);
      if (msg.type === "awareness" && Array.isArray(msg.states)) {
        this.awareness.applyRemote(msg.states);
      }
    } catch {
      // ignore malformed JSON
    }
  }

  private sendAwareness(): void {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) return;
    const local = this.awareness.getLocalState();
    if (!local) return;
    const msg = {
      type: "awareness",
      states: [{ clientId: this.awareness.clientID, state: local }],
    };
    this.ws.send(JSON.stringify(msg));
  }

  setCursor(blockId: string, from: number, to: number): void {
    const current = this.awareness.getLocalState();
    if (!current) return;
    this.awareness.setLocalState({ ...current, cursor: { blockId, from, to } });
  }

  private scheduleReconnect(): void {
    if (this.destroyed) return;
    this.reconnectTimer = setTimeout(() => {
      if (this.pageId && !this.destroyed) {
        const name = this.awareness.getLocalState()?.name ?? "";
        this.doConnect(this.pageId, name);
      }
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
    }, this.reconnectDelay);
  }

  destroy(): void {
    this.destroyed = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    this.ws?.close();
    this.ws = null;
    this.awareness.destroy();
    this.ydoc.destroy();
  }
}
