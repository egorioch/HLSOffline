<template>
  <div>
    <h2 align="center">Play Stream {{ this.suuid }}</h2>
    <div class="container">
      <div class="row">
        <div class="col">
          <video
            id="livestream"
            style="width: 600px;"
            controls
            autoplay
            muted
          ></video>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import Hls from "hls.js";

export default {
  data() {
    return {
      suuid: "H264_AAC", // Значение будет установлено через props или API-запрос
      port: "8083", // Значение будет установлено через props или API-запрос
      suuidMap: ["H264_AAC"], // Значение будет установлено через API-запрос
    };
  },
  mounted() {

    const video = document.getElementById("livestream");
    const videoSrc = `http://localhost:8083/play/hls/${this.suuid}/index.m3u8`;

    if (video.canPlayType("application/vnd.apple.mpegurl")) {
      video.src = videoSrc;
    } else if (Hls.isSupported()) {
      const hls = new Hls({
        autoStartLoad: true,
        debug: true,
        manifestLoadingTimeOut: 20000,
      });
      hls.loadSource(videoSrc);
      hls.attachMedia(video);
    }
  },
};
</script>

<style scoped>
/* Добавьте свои стили, если необходимо */
</style>