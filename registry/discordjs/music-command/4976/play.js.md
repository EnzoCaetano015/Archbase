```javascript
const { SlashCommandBuilder, EmbedBuilder } = require('discord.js');
const { QueryType } = require('discord-player');

module.exports = {
    data: new SlashCommandBuilder()
        .setName('play')
        .setDescription('Adds a track to the server queue.')
        .addStringOption(option => option
            .setName('search')
            .setDescription('Track name or URL.')
            .setRequired(true)
            .setMinLength(2)),

    async execute(interaction) {
        const voiceChannel = interaction.member?.voice?.channel;
        if (!interaction.inGuild() || !voiceChannel) {
            await interaction.reply({ content: 'Join a server voice channel first.', ephemeral: true });
            return;
        }

        const player = interaction.client.player;
        const existingQueue = player.getQueue(interaction.guildId);
        if (existingQueue?.connection && existingQueue.connection.channel.id !== voiceChannel.id) {
            await interaction.reply({ content: 'Join the bot voice channel first.', ephemeral: true });
            return;
        }

        await interaction.deferReply();
        const result = await player.search(interaction.options.getString('search', true), {
            requestedBy: interaction.user,
            searchEngine: QueryType.AUTO,
        });
        if (!result.tracks.length) {
            await interaction.editReply({ content: 'No tracks found.' });
            return;
        }

        const queue = existingQueue || await player.createQueue(interaction.guild, {
            metadata: { channel: interaction.channel },
        });
        const track = result.tracks[0];
        await queue.addTrack(track);
        if (!queue.connection) await queue.connect(voiceChannel);
        if (!queue.playing) await queue.play();

        const embed = new EmbedBuilder()
            .setColor('Green')
            .setTitle('Track added.')
            .setDescription(track.title)
            .setURL(track.url);
        await interaction.editReply({ embeds: [embed] });
    },
};
```
