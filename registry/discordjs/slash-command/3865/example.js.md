```javascript
const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');

module.exports = {
    data: new SlashCommandBuilder()
        .setName('example')
        .setDescription('Demonstrates the command module contract.'),

    async execute(interaction) {
        const embed = new EmbedBuilder()
            .setColor('Green')
            .setTitle('Command completed.')
            .setDescription('Each invocation creates its own response.');
        await interaction.reply({ embeds: [embed] });
    },
};
```
