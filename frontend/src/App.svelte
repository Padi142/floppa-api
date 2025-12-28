<script>
  import { onMount } from "svelte";

  let animals = [];
  let counts = {};

  onMount(async () => {
    try {
      const res = await fetch("/api/animals");
      animals = await res.json();
    } catch (e) {
      console.error("Failed to load animals:", e);
    }
  });

  async function getCount(endpoint) {
    try {
      const res = await fetch(`/${endpoint}/count`);
      const data = await res.json();
      counts[endpoint] = data.count;
      counts = counts; // trigger reactivity
    } catch (e) {
      console.error(e);
      counts[endpoint] = "Error :(";
      counts = counts;
    }
  }
</script>

<div
  class="min-h-screen bg-gradient-to-br from-red-500 via-yellow-500 to-purple-600 bg-[length:400%_400%] animate-gradient-bg font-comic text-blue-700 overflow-x-hidden"
>
  <main class="text-center p-5 relative">
    <div
      class="bg-red-600 text-white p-2 overflow-hidden whitespace-nowrap border-4 border-dashed border-blue-700"
    >
      <h1 class="animate-rainbow text-2xl font-bold">
        dobry den vitajte na stranke padisoft enterprises sro, nas startup ktory
        je buducnostou API obrazkov zvieratiek
      </h1>
    </div>

    <img
      src="https://media.tenor.com/EFDwfjT2GuQAAAAd/spinning-cat.gif"
      class="absolute w-[100px] z-10 top-5 left-5 animate-float"
      alt="Spinning Cat"
    />
    <img
      src="https://media.tenor.com/El89itaAWsIAAAAj/maxwell.gif"
      class="absolute w-[100px] z-10 top-5 right-5 animate-float-reverse"
      alt="Maxwell"
    />

    <div class="content">
      <div
        class="border-8 border-[ridge] border-lime-500 bg-pink-300 p-5 my-5 mx-auto max-w-xl -skew-x-6"
      >
        <h2
          class="underline decoration-wavy decoration-red-600 text-5xl -rotate-6 inline-block mb-4"
        >
          nas tym
        </h2>
        <div class="team-members">
          <div class="text-2xl m-2">
            <span class="text-green-700 drop-shadow-[2px_2px_0px_white]"
              >matyas krejza</span
            >
          </div>
          <div class="text-2xl m-2">
            <span class="text-green-700 drop-shadow-[2px_2px_0px_white]"
              >matej olexa</span
            >
          </div>
        </div>
      </div>

      <div class="flex flex-wrap justify-center gap-5">
        <h2 class="animate-blink text-red-600 text-4xl w-full">
          Nase uzasne funkcie!!!
        </h2>

        {#each animals as animal, i}
          <div
            class="border-4 border-dotted border-fuchsia-500 bg-cyan-300 p-5 w-[300px] rounded-3xl shadow-[10px_10px_0px_black] relative z-10"
            class:animate-border-rotate={i % 2 === 0}
            class:animate-border-rotate-reverse={i % 2 === 1}
          >
            <h3 class="text-xl font-bold mb-2">{animal.title}</h3>
            <p class="mb-4">{animal.description}</p>
            <a
              href="/{animal.endpoint}"
              target="_blank"
              class="inline-block bg-orange-500 border-4 border-[outset] border-red-600 text-xl p-2 text-black font-bold hover:scale-110 transition-transform animate-pulse"
              >&gt;&gt;&gt; Ziskaj &lt;&lt;&lt;</a
            >
            <div class="mt-3">
              <button
                on:click={() => getCount(animal.endpoint)}
                class="inline-block bg-lime-400 border-4 border-[outset] border-green-600 text-sm p-1 text-black font-bold hover:scale-110 transition-transform"
                >Pocet?</button
              >
              {#if counts[animal.endpoint] !== undefined}
                <span class="text-lg font-bold text-red-600 bg-yellow-300 p-1 border-2 border-black ml-2">
                  {counts[animal.endpoint]}
                </span>
              {/if}
            </div>
          </div>
        {/each}
      </div>

      <div class="mt-12 flex justify-around">
        <img
          src="https://media.tenor.com/EFDwfjT2GuQAAAAd/spinning-cat.gif"
          class="w-20 animate-spin"
          alt="Spinning"
        />
        <img
          src="https://media.tenor.com/EFDwfjT2GuQAAAAd/spinning-cat.gif"
          class="w-20 animate-spin direction-reverse"
          alt="Spinning"
        />
      </div>
    </div>
  </main>
</div>

<style>
  /* Custom animations that are hard to do with just utility classes without config */
  @keyframes gradientBG {
    0% {
      background-position: 0% 50%;
    }
    50% {
      background-position: 100% 50%;
    }
    100% {
      background-position: 0% 50%;
    }
  }
  .animate-gradient-bg {
    animation: gradientBG 15s ease infinite;
  }

  @keyframes rainbow {
    0% {
      color: white;
    }
    25% {
      color: yellow;
    }
    50% {
      color: lime;
    }
    75% {
      color: cyan;
    }
    100% {
      color: white;
    }
  }
  .animate-rainbow {
    animation: rainbow 2s linear infinite;
  }

  @keyframes float {
    0% {
      transform: translateY(0px) rotate(0deg);
    }
    50% {
      transform: translateY(-20px) rotate(10deg);
    }
    100% {
      transform: translateY(0px) rotate(0deg);
    }
  }
  .animate-float {
    animation: float 3s ease-in-out infinite;
  }
  .animate-float-reverse {
    animation: float 3s ease-in-out infinite reverse;
  }

  @keyframes borderRotate {
    0% {
      border-color: magenta;
    }
    25% {
      border-color: cyan;
    }
    50% {
      border-color: lime;
    }
    75% {
      border-color: yellow;
    }
    100% {
      border-color: magenta;
    }
  }
  .animate-border-rotate {
    animation: borderRotate 2s linear infinite;
  }
  .animate-border-rotate-reverse {
    animation: borderRotate 2s linear infinite reverse;
  }

  @keyframes blinker {
    50% {
      opacity: 0;
    }
  }
  .animate-blink {
    animation: blinker 1s linear infinite;
  }

  .direction-reverse {
    animation-direction: reverse;
  }

  /* Font fallback */
  .font-comic {
    font-family: "Comic Sans MS", "Chalkboard SE", "Comic Neue", sans-serif;
  }
</style>
