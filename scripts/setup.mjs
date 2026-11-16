import {spawnSync} from 'node:child_process';
import {root} from './go-tool.mjs';
process.chdir(root);
function run(command,args){const result=spawnSync(command,args,{stdio:'inherit',shell:process.platform==='win32'&&command==='npm'});if(result.error)throw result.error;if(result.status!==0)process.exit(result.status??1);}
run(process.execPath,['scripts/setup-go.mjs']);
run('npm',['--prefix','agent-service','ci']);run('npm',['--prefix','frontend','ci']);
run(process.execPath,['scripts/go.mjs','mod','download']);
run(process.execPath,['scripts/configure-env.mjs']);
console.log('Compiling Gin once during setup; a cold Go cache can take several minutes.');
run(process.execPath,['scripts/build-go.mjs']);
