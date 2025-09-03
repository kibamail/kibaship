const fs = require('fs');
const path = require('path');

function parseMultipleJsonObjects(filepath) {
  const content = fs.readFileSync(filepath, 'utf8');
  
  // Split by closing brace followed by newline and opening brace
  const rawObjects = content.trim().split('}\n{');
  
  const objects = rawObjects.map((obj, index, arr) => {
    // Add back the braces that were removed by split
    if (index === 0 && arr.length > 1) {
      return obj + '}';
    } else if (index === arr.length - 1 && arr.length > 1) {
      return '{' + obj;
    } else if (arr.length > 1) {
      return '{' + obj + '}';
    } else {
      // Single object case
      return obj;
    }
  }).map(obj => {
    try {
      return JSON.parse(obj.trim());
    } catch (error) {
      console.error('Error parsing JSON object:', error);
      console.error('Object content:', obj);
      throw error;
    }
  });
  
  return objects;
}

// Parse both files
try {
  console.log('Parsing disks.txt...');
  const disks = parseMultipleJsonObjects('./samples/disks.txt');
  console.log(`Found ${disks.length} disk objects`);
  
  console.log('Parsing addresses.txt...');
  const addresses = parseMultipleJsonObjects('./samples/addresses.txt');
  console.log(`Found ${addresses.length} address objects`);
  
  // Save as proper JSON arrays
  fs.writeFileSync('./samples/disks.json', JSON.stringify(disks, null, 2));
  fs.writeFileSync('./samples/addresses.json', JSON.stringify(addresses, null, 2));
  
  console.log('✅ Successfully converted to valid JSON files:');
  console.log('  - samples/disks.json');
  console.log('  - samples/addresses.json');
  
  // Show sample of first object from each
  console.log('\nSample disk object:');
  console.log(JSON.stringify(disks[0], null, 2));
  
  console.log('\nSample address object:');
  console.log(JSON.stringify(addresses[0], null, 2));
  
} catch (error) {
  console.error('❌ Error:', error.message);
  process.exit(1);
}