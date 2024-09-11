<template>
  <div class="video-bar">
    <div v-for="(vid, idx) in this.suuidMap">
      <button class="video-button" @click="changeVideoSource(vid)">
        {{ idx }}, {{ vid }}
      </button>
    </div>

  </div>
  <div>
    <h2 align="center">Play Stream {{ this.suuid }}</h2>
    <div class="container">
      <div class="row">
        <div class="col">
          <video
              style="width: 600px;"
              ref="livestream"
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
      suuid: "south_develop_room",
      port: "8083",
      suuidMap: ["north_develop_room", "south_develop_room", "kitchen"],
      activeVideos: [],
      videosrc: ""
    };
  },
  mounted() {
    this.videosrc = `http://localhost:8083/play/hls/${this.suuid}/index.m3u8`;
    this.setupVideo();
  },
  methods: {
    setupVideo() {
      const video = this.$refs.livestream;
      const videoSrc = this.videosrc;

      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = videoSrc;
      } else if (Hls.isSupported()) {
        if (this.hls) {
          this.hls.destroy();
        }
        this.hls = new Hls({
          autoStartLoad: true,
          debug: true,
          manifestLoadingTimeOut: 20000,
        });
        this.hls.loadSource(videoSrc);
        this.hls.attachMedia(video);
      } else {
        console.error('Ваш браузер не поддерживает HLS.');
      }
    },
    changeVideoSource(suuid) {
      this.suuid = suuid;
      // `videosrc` обновится автоматически через watcher
    },
  },
  watch: {
    suuid(newSuuid, oldSuuid) {
      this.videosrc = `http://localhost:8083/play/hls/${newSuuid}/index.m3u8`;
    },
    videosrc(newSrc, oldSrc) {
      this.setupVideo();
    },
  },
};
</script>

<style scoped>
.video-bar {
  display: flex;
  width: 100%;
  height: 50px;
  justify-content: center;
}
</style>