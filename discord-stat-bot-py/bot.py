# This example requires the 'members' and 'message_content' privileged intents to function.

import asyncio
import discord
import psutil
import platform
import datetime
import os
import shutil
from discord.ext import commands, tasks

intents = discord.Intents.default()
intents.guilds = True
intents.dm_messages = False
intents.emojis = False
intents.bans = False
intents.message_content = True

bot = commands.Bot(command_prefix='?', intents=intents)
bot.process = psutil.Process()

bot.stats_message = None
channel = None

@bot.event
async def on_ready():
    guild = bot.get_guild(1353806073999396986)
    channel = bot.get_channel(1355803790774767646)
    print(f'Logged in as {bot.user} (ID: {bot.user.id})')
    if not hasattr(bot, 'uptime'):  # Track Uptime
        bot.uptime = datetime.datetime.utcnow()
    message_edit_task.start()
    print('------')
    async for message in channel.history(limit=10):
        if message.author.id == bot.user.id:
            bot.stats_message = message
            break
        
    if not bot.stats_message:
        cpu = f'{round(bot.process.cpu_percent() / psutil.cpu_count(), 1)}% ({psutil.cpu_count()} core/s)'
        totalDisk, usedDisk, freeDisk = shutil.disk_usage("/")
        
        disk = f"Total: {totalDisk/ (1024 ** 3):.2f}\nUsed: {usedDisk/ (1024 ** 3):.2f}\nFree: {freeDisk/ (1024 ** 3):.2f}"
        used = round(psutil.virtual_memory()[3] / 1024**2)
        total = round(psutil.virtual_memory().total / 1024**2)
        ram = f'Bot: {round(bot.process.memory_full_info().rss / 1024**2)}MB\nGlobal usage: {used}MB/{total}MB ({total-used}MB free)'
        pythoninfo = f"Using python `{platform.python_version()}`"
        if os.name == "posix":
            data = platform.freedesktop_os_release()
            distro = f"Running on {data['NAME']}"
        else:
            distro = "Running on some windows machine..."
        delta = datetime.datetime.utcnow() - bot.uptime
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
        embed.add_field(name="Disk", value=disk, inline=False)
        embed.add_field(name="OS", value=distro, inline=True)
        embed.add_field(name="Python", value=pythoninfo, inline=True)
        embed.add_field(name="Guilds:Users", value=f'{len(bot.guilds)}:{len(bot.users)}', inline=False)
        embed.add_field(name="Uptime", value=f"I have been running for {uptime.format(d=days, h=hours, m=minutes, s=seconds)}", inline=False)
        bot.stats_message = await channel.send(embed=embed)

@tasks.loop(seconds=3)
async def message_edit_task():
    if bot.stats_message != None:
        cpu = f'{round(bot.process.cpu_percent() / psutil.cpu_count(), 1)}% ({psutil.cpu_count()} core/s)'
        totalDisk, usedDisk, freeDisk = shutil.disk_usage("/")
        
        disk = f"Total: {totalDisk/ (1024 ** 3):.2f}\nUsed: {usedDisk/ (1024 ** 3):.2f}\nFree: {freeDisk/ (1024 ** 3):.2f}"
        used = round(psutil.virtual_memory()[3] / 1024**2)
        total = round(psutil.virtual_memory().total / 1024**2)
        ram = f'Bot: {round(bot.process.memory_full_info().rss / 1024**2)}MB\nGlobal usage: {used}MB/{total}MB ({total-used}MB free)'
        pythoninfo = f"Using python `{platform.python_version()}`"
        if os.name == "posix":
            data = platform.freedesktop_os_release()
            distro = f"Running on {data['NAME']}"
        else:
            distro = "Running on some windows machine..."
        delta = datetime.datetime.utcnow() - bot.uptime
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
        embed.add_field(name="Disk", value=disk, inline=False)
        embed.add_field(name="OS", value=distro, inline=True)
        embed.add_field(name="Python", value=pythoninfo, inline=True)
        embed.add_field(name="Guilds:Users", value=f'{len(bot.guilds)}:{len(bot.users)}', inline=False)
        embed.add_field(name="Uptime", value=f"I have been running for {uptime.format(d=days, h=hours, m=minutes, s=seconds)}", inline=False)
        await bot.stats_message.edit(embed=embed)


bot.run(open("token.txt", "r").readline())