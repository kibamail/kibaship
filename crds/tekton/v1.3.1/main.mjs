import fs from 'fs/promises'
import path from 'path'
import yaml from 'yaml'
import {kebabCase} from 'change-case'

async function main() {
    const yml = fs.readFile(path.join(process.cwd(), 'main.yaml'), 'utf-8')

    const data = yaml.parseAllDocuments(await yml)

    console.dir(data.length, {depth: null})
    for (let x = 0; x < data.length; x++) {
        const document = data[x]

        const {metadata, kind} = document.toJSON()

        const kindName = kebabCase(kind)
        const idx = x + 1
        const fileName = `${idx > 9 ? idx : '0' + idx}-${metadata.name}-${kindName}.yaml`

        
        await fs.writeFile(path.join(process.cwd(), fileName), document.toString())
        
        console.log(`Written file: ${fileName}`)
    }
}

main().catch(console.error)
