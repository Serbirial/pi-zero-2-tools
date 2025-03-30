# This example requires the 'members' and 'message_content' privileged intents to function.

import asyncio
import discord
import psutil
import platform
import datetime
import os
from discord.ext import commands, tasks

intents = discord.Intents.default()
intents.members = False
intents.dm_messages = False
intents.emojis = False
intents.bans = False
intents.presences = False
intents.message_content = True

bot = commands.Bot(command_prefix='?', intents=intents)
bot.process = psutil.Process()
channel = bot.get_channel(1355803790774767646)

@bot.event
async def on_ready():
    print(f'Logged in as {bot.user} (ID: {bot.user.id})')
    if not hasattr(bot, 'uptime'):  # Track Uptime
        bot.uptime = datetime.datetime.utcnow()
    print('------')


@bot.command()
async def test(ctx):
    """None"""
    pass

@tasks.loop(seconds=3, reconnect=True)
async def my_background_task():
    cpu = f'{round(bot.process.cpu_percent() / psutil.cpu_count(), 1)}% ({psutil.cpu_count()} core/s)'
    used = round(psutil.virtual_memory()[3] / 1024**2)
    total = round(psutil.virtual_memory().total / 1024**2)
    ram = f'Bot: {round(bot.process.memory_full_info().rss / 1024**2)}MB\nGlobal usage: {used}MB/{total}MB ({total-used}MB free)'
    pythoninfo = f"Using python `{platform.python_version()}`"
    if os.name == "posix":
        data = platform.freedesktop_os_release()
        distro = f"Running on {data['NAME']}"
    else:
        distro = "Running on some windows machine..."
    delta = datetime.utcnow() - bot.uptime
    hours, remainder = divmod(int(delta.total_seconds()), 3600)
    minutes, seconds = divmod(remainder, 60)
    days, hours = divmod(hours, 24)
    if days:
        uptime = '{d} days, {h} hours, {m} minutes, and {s} seconds'
    else:
        uptime = '{h} hours, {m} minutes, and {s} seconds'

    embed = discord.Embed(description="Bot stats")
    embed.add_field(name="RAM", value=ram, inline=False)
    embed.add_field(name="CPU", value=cpu, inline=False)
    embed.add_field(name="OS", value=distro, inline=True)
    embed.add_field(name="Python", value=pythoninfo, inline=True)
    embed.add_field(name="Guilds:Users", value=f'{len(bot.guilds)}:{len(bot.users)}', inline=False)
    embed.add_field(name="Uptime", value=f"I have been running for {uptime.format(d=days, h=hours, m=minutes, s=seconds)}", inline=False)
    await channel.send(embed=embed)


bot.run('token')