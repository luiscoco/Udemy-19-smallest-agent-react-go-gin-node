import {spawnSync} from 'node:child_process';
import {resolve} from 'node:path';
import {mkdirSync} from 'node:fs';
import {goCommand,goEnvironment,root} from './go-tool.mjs';
const go=goCommand();if(!go)throw Error('Run npm run setup first.');
const directory=resolve(root,'backend');mkdirSync(resolve(directory,'bin'),{recursive:true});
const output=resolve(directory,'bin',process.platform==='win32'?'gateway.exe':'gateway');
const result=spawnSync(go,['build','-trimpath','-o',output,'.'],{cwd:directory,env:goEnvironment(),stdio:'inherit'});
if(result.status!==0)process.exit(result.status??1);console.log('Built the Gin gateway for '+process.platform+'.');
