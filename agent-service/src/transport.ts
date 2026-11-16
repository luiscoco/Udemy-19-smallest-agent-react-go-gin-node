import type {IncomingMessage,ServerResponse} from 'node:http';
import {timingSafeEqual} from 'node:crypto';
import {Readable} from 'node:stream';
import {pipeline} from 'node:stream/promises';
import type {AgentConfig} from './config.js';
import type {AgentService} from './agent-service.js';
import {createHandlers} from './handlers.js';

function readBounded(req:IncomingMessage):Promise<Buffer|null>{return new Promise((resolve,reject)=>{
 const chunks:Buffer[]=[];let size=0;
 const cleanup=()=>{req.off('data',data);req.off('end',end);req.off('error',error);req.off('aborted',aborted);};
 const error=(failure:Error)=>{cleanup();reject(failure);};const aborted=()=>error(new Error('Request aborted'));
 const end=()=>{cleanup();resolve(Buffer.concat(chunks));};
 const data=(chunk:Buffer|string)=>{const bytes=Buffer.isBuffer(chunk)?chunk:Buffer.from(chunk);size+=bytes.length;if(size>32768){cleanup();req.resume();resolve(null);}else chunks.push(bytes);};
 req.on('data',data);req.once('end',end);req.once('error',error);req.once('aborted',aborted);
});}
export function createInternalHandler(config:AgentConfig,agent:AgentService,token:string){
 const handlers=createHandlers(config,agent);const active=new Set<AbortController>();
 const json=(res:ServerResponse,status:number,detail:string)=>{res.writeHead(status,{'Content-Type':'application/json'});res.end(JSON.stringify({detail}));};
 async function handle(req:IncomingMessage,res:ServerResponse){
  if(req.url==='/internal/ready'&&req.method==='GET'){res.writeHead(200,{'Content-Type':'application/json'});res.end('{"status":"ok"}');return;}
  const supplied=req.headers['x-agent-service-token'];
  const bytes=Buffer.from(typeof supplied==='string'?supplied:'');const expected=Buffer.from(token);
  if(bytes.length!==expected.length||!timingSafeEqual(bytes,expected)){json(res,401,'Agent service authentication failed.');req.resume();return;}
  if(req.url==='/api/health'&&req.method==='GET'){res.writeHead(200,{'Content-Type':'application/json'});res.end(JSON.stringify({status:'ok',api_key_configured:config.apiKeyConfigured}));return;}
  const mode=req.url==='/api/agent/query'?'query':req.url==='/api/agent/stream'?'stream':null;
  if(!mode||req.method!=='POST'){json(res,404,'Not found.');req.resume();return;}
  const controller=new AbortController();active.add(controller);
  const abort=()=>{if(!res.writableEnded)controller.abort();};req.once('aborted',abort);res.once('close',abort);
  try{
   const body=await readBounded(req);if(body===null){res.setHeader('Connection','close');json(res,413,'Request body is too large.');return;}
   const headers=new Headers();for(const [name,value]of Object.entries(req.headers))if(value!==undefined)headers.set(name,Array.isArray(value)?value.join(','):value);
   const request=new Request(`http://${req.headers.host||'127.0.0.1'}${req.url}`,{method:'POST',headers,body:body.toString('utf8'),signal:controller.signal});
   const response=await handlers[mode](request);
   res.writeHead(response.status,Object.fromEntries(response.headers));res.flushHeaders();
   if(response.body)await pipeline(Readable.fromWeb(response.body as never),res);else res.end();
  }catch{controller.abort();if(!res.headersSent)json(res,500,'The agent service could not process this request.');else if(!res.writableEnded)res.end();}
  finally{active.delete(controller);req.off('aborted',abort);res.off('close',abort);}
 }
 return {handle,abortAll(){for(const request of active)request.abort();}};
}
