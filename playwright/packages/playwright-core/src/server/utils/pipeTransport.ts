/**
 * Copyright (c) Microsoft Corporation.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { makeWaitForNextTask } from "./task";

// Secure memory sanitization helper
function secureBufferClear(buffer: Buffer): void {
  if (buffer && buffer.length > 0) {
    buffer.fill(0);
  }
}

interface WritableStream {
  write(data: Buffer): void;
}

interface ReadableStream {
  on(event: "data", callback: (b: Buffer) => void): void;
  on(event: "close", callback: () => void): void;
}

interface ClosableStream {
  close(): void;
}

export class PipeTransport {
  private _pipeWrite: WritableStream;
  private _data = Buffer.from([]);
  private _waitForNextTask = makeWaitForNextTask();
  private _closed = false;
  private _bytesLeft = 0;
  private static readonly LENGTH_HEADER_SIZE = 4; // 32-bit unsigned integer

  onmessage?: (message: string) => void;
  onclose?: () => void;

  private _endian: "be" | "le";
  private _closeableStream: ClosableStream | undefined;

  constructor(
    pipeWrite: WritableStream,
    pipeRead: ReadableStream,
    closeable?: ClosableStream,
    endian: "be" | "le" = "le"
  ) {
    this._pipeWrite = pipeWrite;
    this._endian = endian;
    this._closeableStream = closeable;
    pipeRead.on("data", (buffer) => this._dispatch(buffer));
    pipeRead.on("close", () => {
      this._closed = true;
      if (this.onclose) this.onclose();
    });
    this.onmessage = undefined;
    this.onclose = undefined;
  }

  send(message: string) {
    if (this._closed) throw new Error("Pipe has been closed");
    const data = Buffer.from(message, "utf-8");
    const dataLength = Buffer.alloc(4);
    if (this._endian === "be") dataLength.writeUInt32BE(data.length, 0);
    else dataLength.writeUInt32LE(data.length, 0);
    this._pipeWrite.write(dataLength);
    this._pipeWrite.write(data);
  }

  close() {
    // Let it throw.
    this._closeableStream!.close();
  }

  _dispatch(buffer: Buffer) {
    this._data = Buffer.concat([this._data, buffer]);
    while (true) {
      if (!this._bytesLeft && this._data.length < 4) {
        // Need more data.
        break;
      }

      if (!this._bytesLeft) {
        this._bytesLeft =
          this._endian === "be"
            ? this._data.readUInt32BE(0)
            : this._data.readUInt32LE(0);
        // Zero out the consumed length header
        this._data.fill(0, 0, 4);
        this._data = this._data.slice(4);
      }

      if (!this._bytesLeft || this._data.length < this._bytesLeft) {
        // Need more data.
        break;
      }

      const messageBuffer = this._data.slice(0, this._bytesLeft);
      // Zero out the consumed portion of _data
      this._data.fill(0, 0, this._bytesLeft);
      this._data = this._data.slice(this._bytesLeft);
      this._bytesLeft = 0;

      this._waitForNextTask(() => {
        if (this.onmessage) {
          const messageString = messageBuffer.toString("utf-8");
          try {
            this.onmessage(messageString);
          } finally {
            // Securely clear the message buffer after processing
            secureBufferClear(messageBuffer);
          }
        }
      });
    }
  }
}
