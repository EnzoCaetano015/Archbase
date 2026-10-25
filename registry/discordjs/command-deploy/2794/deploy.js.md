```javascript
const { REST, Routes } = require('discord.js');
const fs = require('node:fs');
const path = require('node:path');

async function deployCommands() {
    const token = process.env.DISCORD_TOKEN;
    const applicationId = process.env.DISCORD_APPLICATION_ID;
    if (!token || !applicationId) throw new Error('Discord deployment configuration is required.');

    const commandsPath = path.join(__dirname, 'commands');
    const commands = [];
    const names = new Set();
    for (const file of fs.readdirSync(commandsPath).filter(file => file.endsWith('.js')).sort()) {
        const command = require(path.join(commandsPath, file));
        if (command.ignore) continue;
        if (!command.data?.name || typeof command.data.toJSON !== 'function' || typeof command.execute !== 'function') {
            throw new Error(`Invalid command module: ${file}`);
        }
        if (names.has(command.data.name)) throw new Error(`Duplicate command name: ${command.data.name}`);
        names.add(command.data.name);
        commands.push(command.data.toJSON());
    }

    const rest = new REST({ version: '10' }).setToken(token);
    await rest.put(Routes.applicationCommands(applicationId), { body: commands });
    console.log(`Published ${commands.length} application commands.`);
}

deployCommands().catch(error => {
    console.error('Command deployment failed.', error);
    process.exitCode = 1;
});
```
