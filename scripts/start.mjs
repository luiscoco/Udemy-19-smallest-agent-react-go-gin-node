import {spawn} from 'node:child_process';
import {resolve} from 'node:path';
import {root} from './go-tool.mjs';
process.chdir(root);const dev=process.argv[2]!=='production';const children=[];let stopping=false;
function stop(code=0){if(stopping)return;stopping=true;for(const child of children){if(process.platform==='win32')spawn('taskkill',['/pid',String(child.pid),'/T','/F'],{stdio:'ignore'});else child.kill('SIGTERM');}setTimeout(()=>process.exit(code),700);}
function launch(command,args,options={}){const child=spawn(command,args,{cwd:root,stdio:'inherit',shell:process.platform==='win32'&&command==='npm',...options});children.push(child);child.on('error',error=>{console.error(error.message);stop(1);});child.on('exit',code=>{if(!stopping)stop(code??1);});return child;}
async function ready(url,attempts=180){for(let i=0;i<attempts&&!stopping;i++){try{if((await fetch(url)).ok)return true;}catch{}if(i>0&&i%60===0)console.log('Waiting for service readiness; Go may still be compiling.');await new Promise(resolve=>setTimeout(resolve,500));}return false;}
process.on('SIGINT',()=>stop());process.on('SIGTERM',()=>stop());
if(dev)launch('npm',['--prefix','agent-service','run','dev']);else launch(process.execPath,['agent-service/dist/server.js']);
if(!await ready('http://127.0.0.1:9401/internal/ready')){console.error('Node agent service did not start on port 9401. Run npm run setup first.');stop(1);}
if(!stopping){const gatewayEnv={...process.env};for(const name of Object.keys(gatewayEnv))if(name.startsWith('ANTHROPIC_')||name.startsWith('CLAUDE_'))delete gatewayEnv[name];
 if(dev)launch(process.execPath,['scripts/go.mjs','run','.'],{env:gatewayEnv});else launch(resolve(root,'backend/bin',process.platform==='win32'?'gateway.exe':'gateway'),[],{cwd:resolve(root,'backend'),env:gatewayEnv});
 if(!await ready('http://127.0.0.1:9400/api/health',1200)){console.error('Gin did not become ready. Check that the service tokens match; default API port is 9400.');stop(1);}
 if(!stopping){if(dev)launch('npm',['--prefix','frontend','run','dev']);else console.log('Open http://127.0.0.1:9400 (React + Gin + Node agent service).');}
}
