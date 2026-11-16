import {existsSync,mkdirSync} from 'node:fs';
import {resolve,dirname} from 'node:path';
import {fileURLToPath} from 'node:url';
import {spawnSync} from 'node:child_process';
export const root=resolve(dirname(fileURLToPath(import.meta.url)),'..');
export const localGo=resolve(root,'.tools/go/bin',process.platform==='win32'?'go.exe':'go');
export function goCommand(){if(existsSync(localGo)&&existsSync(resolve(root,'.tools/go/.lesson19-ready')))return localGo;const result=spawnSync('go',['version'],{encoding:'utf8'});if(result.status===0&&/go1\.(?:2[7-9]|[3-9]\d)\./.test(result.stdout))return 'go';return null;}
export function goEnvironment(){for(const name of ['gomodcache','gobuildcache','gopath'])mkdirSync(resolve(root,'.tools',name),{recursive:true});return {...process.env,GOTOOLCHAIN:'local',GOTELEMETRY:'off',CGO_ENABLED:'0',GOMODCACHE:resolve(root,'.tools/gomodcache'),GOCACHE:resolve(root,'.tools/gobuildcache'),GOPATH:resolve(root,'.tools/gopath')};}
