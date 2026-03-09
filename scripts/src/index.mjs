/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
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
import yargs from 'yargs';
import { hideBin } from 'yargs/helpers';

yargs(hideBin(process.argv))
  .scriptName(process.argv.slice(2))
  .commandDir('commands', {
    extensions: ['mjs'],
    visit: (cmd) => cmd.default ?? cmd,
  })
  .usage('Usage: $0 <command> [options]')
  .demandCommand(
    1,
    'Please specify a command. Try --help for a list of commands.',
  )
  .recommendCommands()
  .strict() // only allow defined commands/options
  .help()
  .alias('h', 'help')
  .version()
  .alias('v', 'version')
  .fail((msg, err, y) => {
    if (err) {
      if (process.env.MYCLI_VERBOSE) console.error('[error]', err);
      else console.error(err.message || err);
    } else {
      console.error(msg);
    }
    process.exit(1);
  })
  .parse();
