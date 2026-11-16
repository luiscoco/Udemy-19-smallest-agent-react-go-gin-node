import {spawn} from 'node:child_process';
import {resolve} from 'node:path';
import {goCommand,goEnvironment,root} from './go-tool.mjs';
const command=goCommand();if(!command){console.error('Install Go 1.27 or run npm run setup:go for this edition’s private toolchain.');process.exit(1);}
const child=spawn(command,process.argv.slice(2),{cwd:resolve(root,'backend'),env:goEnvironment(),stdio:'inherit'});
child.on('error',error=>{console.error(error.message);process.exit(1);});child.on('exit',code=>process.exit(code??1));
