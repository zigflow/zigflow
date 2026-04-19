/*
 * Copyright 2025 - 2026 Zigflow authors <https://github.com/zigflow/zigflow/graphs/contributors>
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Import the framework and instantiate it
import Fastify from "fastify";
import fastifyBasicAuth from "@fastify/basic-auth";

const fastify = Fastify({
  logger: true,
});

const authenticate = { realm: "Zigflow" };

fastify.register(fastifyBasicAuth, { validate, authenticate });

// Declare a route
fastify.after(() => {
  fastify.route({
    method: "GET",
    url: "/basic",
    onRequest: fastify.basicAuth,
    handler: async (req, reply) => {
      return { hello: "world" };
    },
  });
});

// Run the server!
try {
  await fastify.listen({ port: 3000, host: "0.0.0.0" });
} catch (err) {
  fastify.log.error(err);
  process.exit(1);
}

async function handler(request, reply) {
  return { hello: "world" };
}

async function validate(username, password, req, reply) {
  if (username !== "zigflow" || password !== "zigflowftw!") {
    return new Error("Unauthenticated");
  }
}
