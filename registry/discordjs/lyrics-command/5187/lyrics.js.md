```javascript
const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { getLyrics } = require('genius-lyrics-api');

module.exports = {
    data: new SlashCommandBuilder()
        .setName('lyrics')
        .setDescription('Looks up lyrics for the current track.'),

    async execute(interaction) {
        if (!interaction.inGuild()) {
            await interaction.reply({ content: 'Use this command in a server.', ephemeral: true });
            return;
        }
        const queue = interaction.client.player.getQueue(interaction.guildId);
        const voiceChannel = interaction.member?.voice?.channel;
        if (!queue?.playing || !queue.current) {
            await interaction.reply({ content: 'No track is playing.', ephemeral: true });
            return;
        }
        if (!voiceChannel || voiceChannel.id !== queue.connection?.channel?.id) {
            await interaction.reply({ content: 'Join the bot voice channel first.', ephemeral: true });
            return;
        }
        const apiKey = process.env.GENIUS_ACCESS_TOKEN;
        if (!apiKey) throw new Error('GENIUS_ACCESS_TOKEN is required.');

        await interaction.deferReply();
        const track = queue.current;
        const lyrics = await getLyrics({
            apiKey,
            title: track.title,
            artist: ' ',
            optimizeQuery: true,
        });
        if (!lyrics || lyrics.length >= 4000) {
            await interaction.editReply({ content: 'No displayable lyrics found.' });
            return;
        }

        const embed = new EmbedBuilder()
            .setColor('Green')
            .setTitle('Track lyrics')
            .setDescription(lyrics)
            .setURL(track.url);
        await interaction.editReply({ embeds: [embed] });
    },
};
```
