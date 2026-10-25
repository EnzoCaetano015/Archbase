```javascript
const { EmbedBuilder } = require('discord.js');

module.exports = function registerPlayerEvents(client) {
    client.player.on('trackStart', async queue => {
        if (!client.player.showNowPlaying) return;
        const channel = queue.metadata?.channel;
        const track = queue.current;
        if (!channel || !track) return;

        const embed = new EmbedBuilder()
            .setColor('Greyple')
            .setTitle('Now playing')
            .setDescription(track.title)
            .setFooter({ text: `Duration: ${track.duration}` })
            .setURL(track.url);
        try {
            await channel.send({ embeds: [embed] });
        } catch (error) {
            console.error('Player notification failed.', error);
        }
    });
};
```
