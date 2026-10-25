```javascript
const { Client, Collection, GatewayIntentBits } = require('discord.js');
const { Player } = require('discord-player');
const fs = require('node:fs');
const path = require('node:path');
const registerPlayerEvents = require('./events/player');

const client = new Client({
    intents: [GatewayIntentBits.Guilds, GatewayIntentBits.GuildVoiceStates],
});
client.commands = new Collection();
client.player = new Player(client);
client.player.showNowPlaying = true;

const commandsPath = path.join(__dirname, 'commands');
for (const file of fs.readdirSync(commandsPath).filter(file => file.endsWith('.js')).sort()) {
    const command = require(path.join(commandsPath, file));
    if (command.ignore) continue;
    if (!command.data?.name || typeof command.data.toJSON !== 'function' || typeof command.execute !== 'function') {
        throw new Error(`Invalid command module: ${file}`);
    }
    if (client.commands.has(command.data.name)) {
        throw new Error(`Duplicate command name: ${command.data.name}`);
    }
    client.commands.set(command.data.name, command);
}

client.once('ready', () => console.log('Bot ready.'));
client.on('interactionCreate', async interaction => {
    if (!interaction.isChatInputCommand() || interaction.replied || interaction.deferred) return;
    const command = client.commands.get(interaction.commandName);
    try {
        if (!command) {
            await interaction.reply({ content: 'Unknown command.', ephemeral: true });
            return;
        }
        await command.execute(interaction);
    } catch (error) {
        console.error('Command execution failed.', error);
        const response = { content: 'The command could not be completed.', embeds: [] };
        try {
            if (interaction.deferred) await interaction.editReply(response);
            else if (interaction.replied) await interaction.followUp({ ...response, ephemeral: true });
            else await interaction.reply({ ...response, ephemeral: true });
        } catch (replyError) {
            console.error('Command error response failed.', replyError);
        }
    }
});

registerPlayerEvents(client);
const token = process.env.DISCORD_TOKEN;
if (!token) throw new Error('DISCORD_TOKEN is required.');
client.login(token).catch(error => {
    console.error('Bot login failed.', error);
    process.exitCode = 1;
});
```
