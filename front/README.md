# Cubik Frontend

A frontend single-page application for drawing and animating Yeelight CubeLite (Matrix) devices. This application provides an intuitive interface for creating visual content to be displayed on the 20x5 LED matrix.

## Overview

This SvelteKit application serves as the control interface for Yeelight Cube devices. It allows users to:

- Draw and design patterns for the LED matrix display
- Create and preview animations
- Send display commands to the backend for device control

The frontend handles only the user interface and payload formation - all actual device communication is handled by the Go backend. This application is designed to run locally without any authentication or authorization.

## Creating a project

If you're seeing this, you've probably already done this step. Congrats!

```sh
# create a new project in the current directory
npx sv create

# create a new project in my-app
npx sv create my-app
```

## Developing

Once you've created a project and installed dependencies with `npm install` (or `pnpm install` or `yarn`), start a development server:

```sh
npm run dev

# or start the server and open the app in a new browser tab
npm run dev -- --open
```

## Building

To create a production version of your app:

```sh
npm run build
```

You can preview the production build with `npm run preview`.

> To deploy your app, you may need to install an [adapter](https://svelte.dev/docs/kit/adapters) for your target environment.
