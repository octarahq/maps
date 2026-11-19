package com.octara.maps.utils

import com.google.gson.Gson
import com.google.gson.GsonBuilder
import com.google.gson.TypeAdapter
import com.google.gson.stream.JsonReader
import com.google.gson.stream.JsonToken
import com.google.gson.stream.JsonWriter
import com.octara.maps.types.Accessibility
import com.octara.maps.types.Departure
import com.octara.maps.types.FavoriteDepartureInfo
import com.octara.maps.types.JourneyQuery
import com.octara.maps.types.JourneyResponse
import com.octara.maps.types.LegType
import com.octara.maps.types.ParentStop
import com.octara.maps.types.Position
import com.octara.maps.types.Route
import com.octara.maps.types.RouteType
import com.octara.maps.types.Stop
import com.octara.maps.types.StopType
import com.octara.maps.types.TimeType
import com.octara.maps.types.TrainPosition
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.net.HttpURLConnection
import java.net.URL
import java.net.URLEncoder

class Api {
    companion object {
        const val BASE_URL = "https://omaps-calc.orionhost.app/api"
    }

    val gson: Gson = GsonBuilder()
        .registerTypeAdapter(StopType::class.java, object : TypeAdapter<StopType>() {
            override fun write(out: JsonWriter, value: StopType?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): StopType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return StopType.UNKNOWN
                }
                val intVal = reader.nextInt()
                return StopType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(RouteType::class.java, object : TypeAdapter<RouteType>() {
            override fun write(out: JsonWriter, value: RouteType?) {
                out.value(value?.value ?: 3)
            }

            override fun read(reader: JsonReader): RouteType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return RouteType.BUS
                }
                val intVal = reader.nextInt()
                return RouteType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(LegType::class.java, object : TypeAdapter<LegType>() {
            override fun write(out: JsonWriter, value: LegType?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): LegType {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return LegType.WALK
                }
                val intVal = reader.nextInt()
                return LegType.fromValue(intVal)
            }
        })
        .registerTypeAdapter(Accessibility::class.java, object : TypeAdapter<Accessibility>() {
            override fun write(out: JsonWriter, value: Accessibility?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): Accessibility {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return Accessibility.UNKNOWN
                }
                val intVal = reader.nextInt()
                return Accessibility.fromValue(intVal)
            }
        })
        .registerTypeAdapter(TrainPosition::class.java, object : TypeAdapter<TrainPosition>() {
            override fun write(out: JsonWriter, value: TrainPosition?) {
                out.value(value?.value ?: 0)
            }

            override fun read(reader: JsonReader): TrainPosition {
                if (reader.peek() == JsonToken.NULL) {
                    reader.nextNull()
                    return TrainPosition.UNKNOWN
                }
                val intVal = reader.nextInt()
                return TrainPosition.fromValue(intVal)
            }
        })
        .create()

    inline fun <reified T> fetchGET(url: String): T {
        val connection = (URL(url).openConnection() as HttpURLConnection).apply {
            requestMethod = "GET"
            connectTimeout = 10_000
            readTimeout = 10_000
        }

        return try {
            val responseCode = connection.responseCode
            if (responseCode !in 200..299) {
                val errText = try { connection.errorStream?.bufferedReader()?.use { it.readText() } } catch (_: Exception) { null }
                throw IllegalStateException("GET $url failed ($responseCode): $errText")
            }

            val type = object : com.google.gson.reflect.TypeToken<T>() {}.type
            connection.inputStream.bufferedReader().use { reader ->
                gson.fromJson<T>(reader, type)
            } ?: throw IllegalStateException("json is null")
        } finally {
            connection.disconnect()
        }
    }

    private data class StopResponse(
        val total: Int,
        val stops: List<ParentStop>?
    )

    suspend fun searchStopsByName(
        name: String,
        limit: Int = 10,
        cpos: Pair<Float, Float>? = null
    ): List<ParentStop> = withContext(Dispatchers.IO) {
        val encodedQuery = URLEncoder.encode(name, "UTF-8")
        val url = if (cpos != null) {
            "$BASE_URL/stops/autocomplete?q=$encodedQuery&maxstops=$limit&lat=${cpos.first}&lon=${cpos.second}"
        } else {
            "$BASE_URL/stops/autocomplete?q=$encodedQuery&maxstops=$limit"
        }

        val response: StopResponse = fetchGET(url)
        response.stops ?: emptyList()
    }

    data class AddressResult(
        val id: String = "",
        val label: String = "",
        val name: String = "",
        val house_number: String? = null,
        val street: String? = null,
        val postcode: String? = null,
        val city: String? = null,
        val country: String? = null,
        val type: String = "",
        val lat: Float = 0f,
        val lon: Float = 0f,
        val distance_m: Double? = null,
        val score: Double = 0.0
    )

    private data class AddressResponse(
        val total: Int,
        val addresses: List<AddressResult>?
    )

    suspend fun searchAddresses(
        query: String,
        limit: Int = 10,
        cpos: Pair<Float, Float>? = null
    ): List<AddressResult> = withContext(Dispatchers.IO) {
        if (query.trim().length < 3) return@withContext emptyList()
        val encodedQuery = URLEncoder.encode(query, "UTF-8")
        val url = if (cpos != null) {
            "$BASE_URL/addresses/search?q=$encodedQuery&limit=$limit&lat=${cpos.first}&lon=${cpos.second}"
        } else {
            "$BASE_URL/addresses/search?q=$encodedQuery&limit=$limit"
        }

        val response: AddressResponse = fetchGET(url)
        response.addresses ?: emptyList()
    }

    data class PlatformInfo(
        val found: Boolean = false,
        val ref: String? = null,
        val level: String? = null,
        val lat: Double? = null,
        val lon: Double? = null,
        val exit_side: String? = null
    )

    data class LegLiveInfo(
        val found: Boolean = false,
        val cancelled: Boolean = false,
        val dep_delay_sec: Int = 0,
        val arr_delay_sec: Int = 0,
        val real_dep: String? = null,
        val real_arr: String? = null,
        val departure_platform: String? = null,
        val arrival_platform: String? = null,
        val is_long_train: Boolean? = null,
        val source: String? = null,
        val arrival_exit_side: String? = null,
        val arrival_osm_ref: String? = null,
        val arrival_level: String? = null,
        val departure_exit_side: String? = null,
        val departure_osm_ref: String? = null,
        val departure_level: String? = null
    )

    private fun routeTypeToOsmMode(routeType: RouteType?): String? = when (routeType) {
        RouteType.METRO -> "subway"
        RouteType.TRAIN -> "train"
        RouteType.TRAM -> "tram"
        else -> null
    }

    suspend fun getOSMPlatform(
        lat: Double,
        lon: Double,
        ref: String? = null,
        fromLat: Double? = null,
        fromLon: Double? = null,
        allowNearest: Boolean = false,
        headsign: String? = null,
        line: String? = null,
        mode: String? = null,
        nextLat: Double? = null,
        nextLon: Double? = null
    ): PlatformInfo? = withContext(Dispatchers.IO) {
        val params = mutableListOf("lat=$lat", "lon=$lon")
        if (!ref.isNullOrBlank()) {
            params.add("ref=${URLEncoder.encode(ref, "UTF-8")}")
        }
        if (fromLat != null && fromLon != null) {
            params.add("from_lat=$fromLat")
            params.add("from_lon=$fromLon")
        }
        if (nextLat != null && nextLon != null) {
            params.add("next_lat=$nextLat")
            params.add("next_lon=$nextLon")
        }
        if (allowNearest) {
            params.add("allow_nearest=true")
        }
        if (!headsign.isNullOrBlank()) {
            params.add("headsign=${URLEncoder.encode(headsign, "UTF-8")}")
        }
        if (!line.isNullOrBlank()) {
            params.add("line=${URLEncoder.encode(line, "UTF-8")}")
        }
        if (!mode.isNullOrBlank()) {
            params.add("mode=${URLEncoder.encode(mode, "UTF-8")}")
        }
        val url = "$BASE_URL/osm/platform?${params.joinToString("&")}"
        runCatching { fetchGET<PlatformInfo>(url) }.getOrNull()
    }

    suspend fun getLiveLegInfo(leg: com.octara.maps.types.TripLeg): LegLiveInfo? = withContext(Dispatchers.IO) {
        val rid = leg.route?.rid?.takeIf { it.isNotBlank() } ?: return@withContext null
        val schedDep = leg.scheduledDepartureTime ?: leg.departureTime
        val schedArr = leg.scheduledArrivalTime ?: leg.arrivalTime
        val prevStop = leg.intermediateStops?.lastOrNull()?.takeIf { it.lat != 0.0 || it.lon != 0.0 }
        val prevLat = prevStop?.lat ?: leg.fromLat.takeIf { it != 0.0 }
        val prevLon = prevStop?.lon ?: leg.fromLon.takeIf { it != 0.0 }
        val mode = routeTypeToOsmMode(leg.route?.routeType)
        val params = mutableListOf(
            "rid=${URLEncoder.encode(rid, "UTF-8")}",
            "from_sid=${URLEncoder.encode(leg.fromSID, "UTF-8")}",
            "to_sid=${URLEncoder.encode(leg.toSID, "UTF-8")}"
        )
        if (leg.tripIndex > 0L) params.add("trip_index=${leg.tripIndex}")
        if (schedDep.isNotBlank()) params.add("sched_dep=${URLEncoder.encode(schedDep, "UTF-8")}")
        if (schedArr.isNotBlank()) params.add("sched_arr=${URLEncoder.encode(schedArr, "UTF-8")}")
        if (!leg.headsign.isNullOrBlank()) params.add("headsign=${URLEncoder.encode(leg.headsign, "UTF-8")}")
        if (!leg.route?.shortName.isNullOrBlank()) params.add("line=${URLEncoder.encode(leg.route!!.shortName, "UTF-8")}")
        if (!mode.isNullOrBlank()) params.add("mode=${URLEncoder.encode(mode, "UTF-8")}")
        if (leg.fromLat != 0.0 && leg.fromLon != 0.0) {
            params.add("from_lat=${leg.fromLat}")
            params.add("from_lon=${leg.fromLon}")
        }
        if (leg.toLat != 0.0 && leg.toLon != 0.0) {
            params.add("to_lat=${leg.toLat}")
            params.add("to_lon=${leg.toLon}")
        }
        if (prevLat != null && prevLon != null) {
            params.add("prev_lat=$prevLat")
            params.add("prev_lon=$prevLon")
        }
        val url = "$BASE_URL/transit/leg-live?${params.joinToString("&")}"
        runCatching { fetchGET<LegLiveInfo>(url) }.getOrNull()
    }

    suspend fun resolveLiveArrivalPlatform(
        leg: com.octara.maps.types.TripLeg,
        preloadedLive: LegLiveInfo? = null,
        allowLegLiveFetch: Boolean = true
    ): PlatformInfo? {
        val routeType = leg.route?.routeType
        val isMetroOrTram = routeType == RouteType.METRO || routeType == RouteType.TRAM
        val mode = routeTypeToOsmMode(routeType)
        val lineName = leg.route?.shortName

        val liveInfo = preloadedLive ?: if (allowLegLiveFetch) getLiveLegInfo(leg) else null
        val directArrPlatform = liveInfo?.arrival_platform?.trim()?.takeIf { it.isNotBlank() }
            ?: leg.arrivalPlatform?.trim()?.takeIf { it.isNotBlank() }

        if (liveInfo != null && (!liveInfo.arrival_exit_side.isNullOrBlank() || !directArrPlatform.isNullOrBlank() || !liveInfo.arrival_osm_ref.isNullOrBlank())) {
            val resolvedRef = directArrPlatform ?: liveInfo.arrival_osm_ref?.takeIf { it.length <= 6 }
            if (resolvedRef != null || !liveInfo.arrival_exit_side.isNullOrBlank()) {
                return PlatformInfo(
                    found = true,
                    ref = resolvedRef,
                    level = liveInfo.arrival_level,
                    exit_side = liveInfo.arrival_exit_side?.takeIf { it.isNotBlank() }
                )
            }
        }

        val rid = leg.route?.rid?.takeIf { it.isNotBlank() }
        val fallbackPlatform = if (directArrPlatform == null && !isMetroOrTram && rid != null) {
            val details = runCatching {
                getNextDepartureDetailsOfMultipleStops(listOf(leg.toSID to rid))
            }.getOrNull()
            val cleanStop = leg.toSID.removePrefix("stop_point:IDFM:").removePrefix("IDFM:")
            val cleanRid = rid.removePrefix("IDFM:")
            val byHeadsign = details?.get(cleanStop to cleanRid)
                ?: details?.entries?.firstOrNull { it.key.second == cleanRid }?.value
            val headsign = leg.headsign?.trim()
            val candidates = if (byHeadsign != null) {
                if (!headsign.isNullOrBlank()) {
                    byHeadsign.entries.firstOrNull { it.key.equals(headsign, ignoreCase = true) }?.value
                        ?: byHeadsign.values.flatten()
                } else {
                    byHeadsign.values.flatten()
                }
            } else emptyList()
            candidates.firstNotNullOfOrNull { it.platform?.trim()?.takeIf(String::isNotBlank) }
        } else null

        val livePlatform = directArrPlatform ?: fallbackPlatform
        if (livePlatform == null && !isMetroOrTram) {
            return null
        }

        if (leg.toLat == 0.0 && leg.toLon == 0.0) {
            return livePlatform?.let { PlatformInfo(found = true, ref = it) }
        }
        val prevStop = leg.intermediateStops?.lastOrNull()?.takeIf { it.lat != 0.0 || it.lon != 0.0 }
        val fromLat = (prevStop?.lat ?: leg.fromLat).takeIf { it != 0.0 }
        val fromLon = (prevStop?.lon ?: leg.fromLon).takeIf { it != 0.0 }
        val osmInfo = getOSMPlatform(
            lat = leg.toLat,
            lon = leg.toLon,
            ref = livePlatform,
            fromLat = fromLat,
            fromLon = fromLon,
            allowNearest = isMetroOrTram,
            headsign = leg.headsign,
            line = lineName,
            mode = mode
        )
        return when {
            osmInfo != null && osmInfo.found -> {
                val resolvedRef = livePlatform ?: osmInfo.ref?.takeIf { it.length <= 6 }
                osmInfo.copy(ref = resolvedRef)
            }
            livePlatform != null -> PlatformInfo(found = true, ref = livePlatform)
            else -> null
        }
    }

    data class IndoorStep(
        val instruction: String = "",
        val kind: String = "walk",
        val distance_m: Int = 0,
        val duration_sec: Int = 0,
        val from_level: String? = null,
        val to_level: String? = null,
        val bearing_deg: Double = 0.0,
        val turn: String = "straight",
        val lat: Double = 0.0,
        val lon: Double = 0.0,
        val end_lat: Double = 0.0,
        val end_lon: Double = 0.0
    )

    data class IndoorRouteResult(
        val found: Boolean = false,
        val total_distance_m: Int = 0,
        val total_duration_sec: Int = 0,
        val steps: List<IndoorStep> = emptyList()
    )

    suspend fun getTransferGuide(
        fromLat: Double,
        fromLon: Double,
        toLat: Double,
        toLon: Double,
        fromRef: String? = null,
        toRef: String? = null,
        fromNearest: Boolean = false,
        toNearest: Boolean = false,
        wheelchair: Boolean = false
    ): IndoorRouteResult? = withContext(Dispatchers.IO) {
        if ((fromLat == 0.0 && fromLon == 0.0) || (toLat == 0.0 && toLon == 0.0)) {
            return@withContext null
        }
        val params = mutableListOf(
            "from_lat=$fromLat",
            "from_lon=$fromLon",
            "to_lat=$toLat",
            "to_lon=$toLon"
        )
        if (!fromRef.isNullOrBlank()) params.add("from_ref=${URLEncoder.encode(fromRef, "UTF-8")}")
        if (!toRef.isNullOrBlank()) params.add("to_ref=${URLEncoder.encode(toRef, "UTF-8")}")
        if (fromNearest) params.add("from_nearest=true")
        if (toNearest) params.add("to_nearest=true")
        if (wheelchair) params.add("wheelchair=true")
        val url = "$BASE_URL/osm/transfer-guide?${params.joinToString("&")}"
        runCatching { fetchGET<IndoorRouteResult>(url) }.getOrNull()
    }

    public data class StopInfoResponse(
        val stop: Stop,
        val departures: List<Departure>?,
        val routes: List<Route>?
    )

    suspend fun getStopById(
        id: String,
    ): StopInfoResponse = withContext(Dispatchers.IO) {
        val url = "$BASE_URL/stops/$id"

        val response: StopInfoResponse = fetchGET(url)
        response
    }

    suspend fun getNextDeparturesOfMultipleStops(list: List<Pair<String, String>>): Map<Pair<String, String>, Map<String, List<Int>>> =
        withContext(Dispatchers.IO) {
            val details = getNextDepartureDetailsOfMultipleStops(list)
            details.mapValues { (_, byHeadsign) ->
                byHeadsign.mapValues { (_, items) -> items.map { it.wait_sec } }
            }
        }

    suspend fun getNextDepartureDetailsOfMultipleStops(list: List<Pair<String, String>>): Map<Pair<String, String>, Map<String, List<FavoriteDepartureInfo>>> =
        withContext(Dispatchers.IO) {
            if (list.isEmpty()) return@withContext emptyMap()
            try {
                val url = "$BASE_URL/nextdepartures?details=true&stops=${list.joinToString(",") { "${it.first}:${it.second}" }}"
                val response: Map<String, Map<String, List<Any>>> = fetchGET(url)
                val result = mutableMapOf<Pair<String, String>, Map<String, List<FavoriteDepartureInfo>>>()

                for ((key, deps) in response) {
                    val clean = key.removePrefix("stop_point:IDFM:").removePrefix("IDFM:")
                    val parts = clean.split(":")
                    if (parts.size >= 2) {
                        val stopId = parts[0]
                        val routeId = parts[1]
                        val sanitizedDeps = deps.mapValues { (_, items) ->
                            items.mapNotNull { item ->
                                when (item) {
                                    is Number -> FavoriteDepartureInfo(wait_sec = item.toInt())
                                    is String -> item.toIntOrNull()?.let { FavoriteDepartureInfo(wait_sec = it) }
                                    is Map<*, *> -> {
                                        val waitSec = (item["wait_sec"] as? Number)?.toInt() ?: 0
                                        val depSec = (item["dep_sec"] as? Number)?.toInt() ?: 0
                                        val schedDepSec = (item["scheduled_dep_sec"] as? Number)?.toInt() ?: 0
                                        val delaySec = (item["delay_sec"] as? Number)?.toInt() ?: 0
                                        val delayMin = (item["delay_minutes"] as? Number)?.toInt() ?: ((delaySec + 30) / 60).coerceAtLeast(0)
                                        val isDelayed = (item["is_delayed"] as? Boolean) ?: (delaySec >= 60)
                                        val timeStr = item["time_string"] as? String
                                        val schedTimeStr = item["scheduled_time_string"] as? String
                                        val platform = item["platform"] as? String
                                        val isLongTrain = item["is_long_train"] as? Boolean
                                        FavoriteDepartureInfo(
                                            wait_sec = waitSec,
                                            dep_sec = depSec,
                                            scheduled_dep_sec = schedDepSec,
                                            delay_sec = delaySec,
                                            delay_minutes = delayMin,
                                            is_delayed = isDelayed,
                                            time_string = timeStr,
                                            scheduled_time_string = schedTimeStr,
                                            platform = platform,
                                            is_long_train = isLongTrain
                                        )
                                    }
                                    else -> null
                                }
                            }
                        }
                        result[stopId to routeId] = sanitizedDeps
                        result["IDFM:$stopId" to "IDFM:$routeId"] = sanitizedDeps
                    }
                }

                for (pair in list) {
                    val cleanPair = pair.first.removePrefix("IDFM:") to pair.second.removePrefix("IDFM:")
                    if (result.containsKey(cleanPair)) {
                        result[pair] = result[cleanPair]!!
                    }
                }
                result
            } catch (_: Exception) {
                emptyMap()
            }
        }

    suspend fun getJourney(query: JourneyQuery): JourneyResponse = withContext(Dispatchers.IO) {
        val params = mutableListOf<Pair<String, String>>()

        when (val from = query.from) {
            is Position.Coordinates -> {
                params.add("from_lat" to from.lat.toString())
                params.add("from_lon" to from.lon.toString())
            }

            is Position.ByStop -> {
                params.add("from_stop" to from.stopId)
            }

            is Position.CurrentLocation -> {
                query.currentGps?.let {
                    params.add("from_lat" to it.first.toString())
                    params.add("from_lon" to it.second.toString())
                } ?: throw IllegalArgumentException("invalid 'from' position: CurrentLocation but currentGps is null")
            }
        }

        when (val to = query.to) {
            is Position.Coordinates -> {
                params.add("to_lat" to to.lat.toString())
                params.add("to_lon" to to.lon.toString())
            }

            is Position.ByStop -> {
                params.add("to_stop" to to.stopId)
            }

            is Position.CurrentLocation -> {
                query.currentGps?.let {
                    params.add("to_lat" to it.first.toString())
                    params.add("to_lon" to it.second.toString())
                } ?: throw IllegalArgumentException("invalid 'to' position: CurrentLocation but currentGps is null")
            }
        }

        if (!query.time.isNullOrBlank()) {
            params.add("time" to query.time)
            params.add("timetype" to if (query.timeType == TimeType.ARRIVAL) "arrival" else "departure")
        }

        if (!query.allowedModes.isNullOrEmpty()) {
            val modesStr = query.allowedModes.mapNotNull { mode ->
                when (mode) {
                    RouteType.TRAM -> "tram"
                    RouteType.METRO -> "metro"
                    RouteType.TRAIN -> "train"
                    RouteType.BUS -> "bus"
                    RouteType.FERRY -> "ferry"
                    RouteType.CABLE_TRAM -> "cabletram"
                    RouteType.AERIAL_LIFT -> "aeriallift"
                    RouteType.FUNICULAR -> "funicular"
                }
            }.joinToString(",")
            if (modesStr.isNotBlank()) {
                params.add("allowedmodes" to modesStr)
            }
        }

        if (query.withBike) {
            params.add("withbike" to "true")
        }
        if (query.wheelchair) {
            params.add("wheelchair" to "true")
        }

        val queryString = params.joinToString("&") { (key, value) ->
            "$key=${URLEncoder.encode(value, "UTF-8")}"
        }

        val url = "$BASE_URL/transit/journey?$queryString"
        fetchGET<JourneyResponse>(url)
    }

    suspend fun getLayerStops(x: Int, y: Int, z: Int): com.octara.maps.types.LayerStopsResponse {
        val url = "$BASE_URL/layers/stops/$x/$y/$z"
        return fetchGET<com.octara.maps.types.LayerStopsResponse>(url)
    }

    suspend fun getLayerRoutesLines(x: Int, y: Int, z: Int): com.octara.maps.types.LayerRoutesLinesResponse {
        val url = "$BASE_URL/layers/routesLines/$x/$y/$z"
        return fetchGET<com.octara.maps.types.LayerRoutesLinesResponse>(url)
    }

    suspend fun getJourney(
        from: Position,
        to: Position,
        time: String? = null,
        timeType: TimeType = TimeType.DEPARTURE,
        allowedModes: List<RouteType>? = null,
        withBike: Boolean = false,
        wheelchair: Boolean = false,
        currentGps: Pair<Double, Double>? = null
    ): JourneyResponse = getJourney(
        JourneyQuery(
            from = from,
            to = to,
            time = time,
            timeType = timeType,
            allowedModes = allowedModes,
            withBike = withBike,
            wheelchair = wheelchair,
            currentGps = currentGps
        )
    )
}

val api = Api()
