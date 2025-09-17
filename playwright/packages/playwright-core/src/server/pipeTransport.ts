/**
 * Copyright 2018 Google Inc. All rights reserved.
 * Modifications copyright (c) Microsoft Corporation.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { makeWaitForNextTask } from '../utils';
import { debugLogger } from './utils/debugLogger';

import type { ConnectionTransport, ProtocolRequest, ProtocolResponse } from './transport';

export class PipeTransport implements ConnectionTransport {
  private _pipeRead: NodeJS.ReadableStream;
  private _pipeWrite: NodeJS.WritableStream;
  private _pendingBuffers: Buffer[] = [];
  private _waitForNextTask = makeWaitForNextTask();
  private _closed = false;
  private _onclose?: (reason?: string) => void;

  onmessage?: (message: ProtocolResponse) => void;

  constructor(pipeWrite: NodeJS.WritableStream, pipeRead: NodeJS.ReadableStream) {
    this._pipeRead = pipeRead;
    this._pipeWrite = pipeWrite;
    pipeRead.on('data', buffer => this._dispatch(buffer));
    pipeRead.on('close', () => {
      this._closed = true;
      if (this._onclose)
        this._onclose.call(null);
    });
    pipeRead.on('error', e => debugLogger.log('error', e));
    pipeWrite.on('error', e => debugLogger.log('error', e));
    this.onmessage = undefined;
  }

  get onclose() {
    return this._onclose;
  }

  set onclose(onclose: undefined | ((reason?: string) => void)) {
    this._onclose = onclose;
    if (onclose && !this._pipeRead.readable)
      onclose();
  }

  send(message: ProtocolRequest) {
    if (this._closed)
      throw new Error('Pipe has been closed');
    this._pipeWrite.write(JSON.stringify(message));
    this._pipeWrite.write('\0');
  }

  close() {
    throw new Error('unimplemented');
  }

  _dispatch(buffer: Buffer) {
    let end = buffer.indexOf('\0');
    if (end === -1) {
      this._pendingBuffers.push(buffer);
      return;
    }
    this._pendingBuffers.push(buffer.slice(0, end));
    let message = Buffer.concat(this._pendingBuffers).toString();

    // Clear pending buffers immediately after concatenation
    this._pendingBuffers = [];

    this._waitForNextTask(() => {
      if (this.onmessage) {
        try {
          this.onmessage.call(null, JSON.parse(message));
        } finally {
          // Clear the message variable immediately after processing
          message = '';
        }
      }
    });

    let start = end + 1;
    end = buffer.indexOf('\0', start);
    while (end !== -1) {
      let message = buffer.toString(undefined, start, end);
      this._waitForNextTask(() => {
        if (this.onmessage) {
          try {
            this.onmessage.call(null, JSON.parse(message));
          } finally {
            // Clear the message variable immediately after processing
            message = '';
          }
        }
      });
      start = end + 1;
      end = buffer.indexOf('\0', start);
    }

    // Only keep the remaining buffer slice, not the entire buffer
    const remainingSlice = buffer.slice(start);
    if (remainingSlice.length > 0) {
      this._pendingBuffers = [remainingSlice];
    } else {
      this._pendingBuffers = [];
    }
  }
}
