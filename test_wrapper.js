const { spawn } = require('child_process');

// Get the sap-odata-mcp-universal path
const odataMcpPath = './sap-odata-mcp-universal';

// Spawn the process
const child = spawn(odataMcpPath, process.argv.slice(2), {
  stdio: ['inherit', 'inherit', 'inherit']
});

// Handle exit
child.on('exit', (code) => {
  process.exit(code);
});
