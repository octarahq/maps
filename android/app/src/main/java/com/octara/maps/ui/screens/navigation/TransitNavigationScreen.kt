package com.octara.maps.ui.screens.navigation

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import androidx.activity.compose.BackHandler
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.slideInVertically
import androidx.compose.animation.slideOutVertically
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.NotificationsActive
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.WifiOff
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableLongStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshotFlow
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.platform.LocalView
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.zIndex
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.LocalLifecycleOwner
import com.octara.maps.managers.ActiveNavigationManager
import com.octara.maps.types.LegType
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.RouteType
import com.octara.maps.ui.components.JourneyLegItem
import com.octara.maps.ui.components.LegNavigationState
import com.octara.maps.utils.Api
import com.octara.maps.utils.LocationManager
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private fun mergeLiveOverride(
    trip: ParsedTrip,
    overrides: Map<Int, Api.LegLiveInfo>
): ParsedTrip {
    if (overrides.isEmpty()) return trip
    val updatedLegs = trip.legs.mapIndexed { idx, leg ->
        val live = overrides[idx] ?: return@mapIndexed leg
        val newDepPlat = live.departure_platform?.takeIf { it.isNotBlank() } ?: leg.departurePlatform
        val newArrPlat = live.arrival_platform?.takeIf { it.isNotBlank() } ?: leg.arrivalPlatform
        val newDepTime = live.real_dep?.takeIf { it.isNotBlank() } ?: leg.departureTime
        val newArrTime = live.real_arr?.takeIf { it.isNotBlank() } ?: leg.arrivalTime
        val newIsDelayed = leg.isDelayed || live.dep_delay_sec >= 60 || live.arr_delay_sec >= 60
        leg.copy(
            departurePlatform = newDepPlat,
            arrivalPlatform = newArrPlat,
            departureTime = newDepTime,
            arrivalTime = newArrTime,
            scheduledDepartureTime = leg.scheduledDepartureTime ?: if (live.real_dep != null) leg.departureTime else null,
            scheduledArrivalTime = leg.scheduledArrivalTime ?: if (live.real_arr != null) leg.arrivalTime else null,
            departureDelaySec = if (live.found) live.dep_delay_sec else leg.departureDelaySec,
            arrivalDelaySec = if (live.found) live.arr_delay_sec else leg.arrivalDelaySec,
            isDelayed = newIsDelayed,
            isLongTrain = live.is_long_train ?: leg.isLongTrain
        )
    }
    return trip.copy(legs = updatedLegs)
}

@Composable
fun TransitNavigationScreen(
    trip: ParsedTrip,
    onBack: () -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current
    val lifecycleOwner = LocalLifecycleOwner.current
    val handleExit: () -> Unit = {
        ActiveNavigationManager.clearActiveNavigation(context)
        onBack()
    }

    BackHandler {
        handleExit()
    }

    val view = LocalView.current
    DisposableEffect(view) {
        val previous = view.keepScreenOn
        view.keepScreenOn = true
        onDispose {
            view.keepScreenOn = previous
        }
    }

    DisposableEffect(Unit) {
        if (LocationManager.hasLocationPermission(context)) {
            LocationManager.setHighAccuracyNavigation(context, true)
        }
        onDispose {
            LocationManager.setHighAccuracyNavigation(context, false)
        }
    }

    var resyncTick by remember { mutableIntStateOf(0) }
    var networkReconnectTick by remember { mutableIntStateOf(0) }
    var isNetworkOnline by remember { mutableStateOf(true) }

    DisposableEffect(context) {
        val cm = context.applicationContext.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager
        if (cm != null) {
            val activeCap = runCatching { cm.getNetworkCapabilities(cm.activeNetwork) }.getOrNull()
            isNetworkOnline = activeCap?.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) == true
            val callback = object : ConnectivityManager.NetworkCallback() {
                override fun onAvailable(network: Network) {
                    isNetworkOnline = true
                    networkReconnectTick++
                }

                override fun onLost(network: Network) {
                    val cap = runCatching { cm.getNetworkCapabilities(cm.activeNetwork) }.getOrNull()
                    isNetworkOnline = cap?.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) == true
                }

                override fun onCapabilitiesChanged(network: Network, networkCapabilities: NetworkCapabilities) {
                    if (networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)) {
                        isNetworkOnline = true
                    }
                }
            }
            runCatching { cm.registerDefaultNetworkCallback(callback) }
            onDispose {
                runCatching { cm.unregisterNetworkCallback(callback) }
            }
        } else {
            onDispose { }
        }
    }

    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                LocationManager.kickLocationRefresh(context)
                resyncTick++
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    LaunchedEffect(trip) {
        ActiveNavigationManager.saveActiveNavigation(context, trip)
    }

    val realLocation by LocationManager.currentLocation.collectAsState()
    val lastFixMillis by LocationManager.lastFixMillis.collectAsState()
    var nowMillis by remember(trip) {
        mutableLongStateOf(System.currentTimeMillis())
    }

    LaunchedEffect(Unit) {
        while (true) {
            delay(1_000L)
            nowMillis = System.currentTimeMillis()
        }
    }

    LaunchedEffect(Unit) {
        while (true) {
            delay(20_000L)
            val ageMs = System.currentTimeMillis() - LocationManager.lastFixMillis.value
            if (ageMs > 25_000L) {
                LocationManager.kickLocationRefresh(context)
            }
        }
    }

    val tracker = remember(trip) { NavigationSessionTracker() }
    var liveLegOverrides by remember(trip) { mutableStateOf<Map<Int, Api.LegLiveInfo>>(emptyMap()) }
    var liveArrivalPlatforms by remember(trip) { mutableStateOf<Map<Int, Api.PlatformInfo>>(emptyMap()) }

    val liveTrip = remember(trip, liveLegOverrides) {
        mergeLiveOverride(trip, liveLegOverrides)
    }

    val platformHints = remember(liveArrivalPlatforms) {
        liveArrivalPlatforms.mapValues { (_, v) ->
            LegPlatformHint(
                ref = v.ref?.takeIf { it.isNotBlank() },
                exitSide = v.exit_side?.takeIf { it == "left" || it == "right" }
            )
        }
    }

    val progress = remember(
        liveTrip,
        realLocation,
        lastFixMillis,
        nowMillis,
        platformHints,
        resyncTick
    ) {
        resolveTrackedNavigationProgress(
            trip = liveTrip,
            location = realLocation,
            gpsFixMillis = lastFixMillis,
            nowMillis = nowMillis,
            tracker = tracker,
            platformHints = platformHints
        )
    }

    LaunchedEffect(progress.activeLegIndex) {
        val activeLeg = liveTrip.legs.getOrNull(progress.activeLegIndex)
        if (activeLeg?.type == LegType.WALK || activeLeg?.type == LegType.TRANSFERT) {
            com.octara.maps.utils.IndoorStepTracker.resetProgress()
            com.octara.maps.utils.IndoorStepTracker.start(context)
        } else {
            com.octara.maps.utils.IndoorStepTracker.stop()
        }
    }

    DisposableEffect(Unit) {
        onDispose {
            com.octara.maps.utils.IndoorStepTracker.stop()
        }
    }

    LaunchedEffect(progress.activeLegIndex, trip, resyncTick, networkReconnectTick) {
        val api = Api()
        val targetIndices = buildList {
            trip.legs.forEachIndexed { idx, l ->
                if (l.type == LegType.TRANSIT && idx >= progress.activeLegIndex) {
                    add(idx)
                }
            }
        }
        if (targetIndices.isEmpty()) return@LaunchedEffect

        while (true) {
            var allComplete = true
            for (idx in targetIndices) {
                val curLeg = mergeLiveOverride(trip, liveLegOverrides).legs.getOrNull(idx) ?: continue
                val liveInfo = api.getLiveLegInfo(curLeg)
                if (liveInfo != null) {
                    liveLegOverrides = liveLegOverrides + (idx to liveInfo)
                }
                val mergedLeg = mergeLiveOverride(trip, liveLegOverrides).legs.getOrNull(idx) ?: curLeg
                val arrInfo = api.resolveLiveArrivalPlatform(
                    leg = mergedLeg,
                    preloadedLive = liveInfo,
                    allowLegLiveFetch = false
                )
                if (arrInfo != null && (arrInfo.found || !arrInfo.ref.isNullOrBlank() || !arrInfo.exit_side.isNullOrBlank())) {
                    val prev = liveArrivalPlatforms[idx]
                    val combined = Api.PlatformInfo(
                        found = true,
                        ref = arrInfo.ref?.takeIf { it.isNotBlank() } ?: prev?.ref,
                        level = arrInfo.level ?: prev?.level,
                        lat = arrInfo.lat ?: prev?.lat,
                        lon = arrInfo.lon ?: prev?.lon,
                        exit_side = arrInfo.exit_side?.takeIf { it == "left" || it == "right" } ?: prev?.exit_side
                    )
                    liveArrivalPlatforms = liveArrivalPlatforms + (idx to combined)
                } else {
                    allComplete = false
                }
                if (mergedLeg.route?.routeType == RouteType.TRAIN && mergedLeg.departurePlatform.isNullOrBlank()) {
                    allComplete = false
                }
            }
            delay(if (allComplete) 20_000L else 5_000L)
        }
    }

    val listState = rememberLazyListState()
    val coroutineScope = rememberCoroutineScope()
    val density = LocalDensity.current
    var isUserScrollingAway by remember { mutableStateOf(false) }
    var lastUserInteractionMillis by remember { mutableLongStateOf(0L) }
    var isProgrammaticScroll by remember { mutableStateOf(false) }

    suspend fun scrollToActiveLeg() {
        if (progress.adjustedLegs.isEmpty()) return
        isProgrammaticScroll = true
        try {
            val stopPos = progress.activeStopPosition
            val offsetPx = if (stopPos != null && stopPos >= 2) {
                with(density) { (120.dp + 40.dp * (stopPos - 2)).roundToPx() }
            } else {
                0
            }
            listState.animateScrollToItem(progress.activeLegIndex, scrollOffset = offsetPx)
        } finally {
            isProgrammaticScroll = false
        }
    }

    LaunchedEffect(listState) {
        snapshotFlow {
            Triple(
                listState.isScrollInProgress,
                listState.firstVisibleItemIndex,
                listState.firstVisibleItemScrollOffset
            )
        }.collect { (inProgress, _, _) ->
            if (inProgress && !isProgrammaticScroll) {
                isUserScrollingAway = true
                lastUserInteractionMillis = System.currentTimeMillis()
            }
        }
    }

    LaunchedEffect(progress.activeLegIndex, progress.activeStopPosition) {
        if (!isUserScrollingAway) {
            scrollToActiveLeg()
        }
    }

    LaunchedEffect(isUserScrollingAway, lastUserInteractionMillis, progress.activeLegIndex) {
        if (isUserScrollingAway) {
            delay(5_000L)
            isUserScrollingAway = false
            scrollToActiveLeg()
        }
    }

    val triggerManualResync: () -> Unit = {
        if (LocationManager.hasLocationPermission(context)) {
            LocationManager.kickLocationRefresh(context)
        }
        tracker.resetForResync()
        resyncTick++
        isUserScrollingAway = false
        coroutineScope.launch {
            scrollToActiveLeg()
        }
    }

    Surface(
        modifier = modifier.fillMaxSize(),
        color = Color.White
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            Column(modifier = Modifier.fillMaxSize()) {

                Surface(
                    color = MaterialTheme.colorScheme.surface,
                    shadowElevation = 4.dp,
                    modifier = Modifier
                        .fillMaxWidth()
                        .zIndex(10f)
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .statusBarsPadding()
                            .padding(horizontal = 12.dp, vertical = 10.dp)
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Surface(
                                shape = CircleShape,
                                color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.7f),
                                modifier = Modifier.size(40.dp)
                            ) {
                                IconButton(
                                    onClick = handleExit,
                                    modifier = Modifier.fillMaxSize()
                                ) {
                                    Icon(
                                        imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                                        contentDescription = "Retour",
                                        tint = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.width(12.dp))

                            Column(modifier = Modifier.weight(1f)) {
                                Text(
                                    text = progress.nextActionTitle,
                                    style = MaterialTheme.typography.titleMedium,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.onSurface,
                                    maxLines = 2,
                                    overflow = TextOverflow.Ellipsis
                                )
                                if (!progress.nextActionSubtitle.isNullOrBlank()) {
                                    Spacer(modifier = Modifier.height(2.dp))
                                    Text(
                                        text = progress.nextActionSubtitle,
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                                        maxLines = 2,
                                        overflow = TextOverflow.Ellipsis
                                    )
                                }
                                if (progress.arrivalTimeText.isNotBlank()) {
                                    Spacer(modifier = Modifier.height(2.dp))
                                    Text(
                                        text = "Arrivée finale ${progress.arrivalTimeText} (${progress.remainingMinutes} min)",
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.SemiBold,
                                        color = MaterialTheme.colorScheme.primary
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.width(8.dp))

                            Surface(
                                shape = CircleShape,
                                color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.7f),
                                modifier = Modifier.size(40.dp)
                            ) {
                                IconButton(
                                    onClick = triggerManualResync,
                                    modifier = Modifier.fillMaxSize()
                                ) {
                                    Icon(
                                        imageVector = Icons.Default.Refresh,
                                        contentDescription = "Resynchroniser GPS et horaires",
                                        tint = MaterialTheme.colorScheme.onSurface
                                    )
                                }
                            }
                        }

                        AnimatedVisibility(visible = !isNetworkOnline || !progress.gpsSynced) {
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFFFF8E1),
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp)
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 5.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Default.WifiOff,
                                        contentDescription = null,
                                        tint = Color(0xFFEF6C00),
                                        modifier = Modifier.size(16.dp)
                                    )
                                    val statusText = when {
                                        !isNetworkOnline && !progress.gpsSynced -> "Hors ligne et GPS perdu — resync auto"
                                        !isNetworkOnline -> "Connexion perdue — horaires en cache"
                                        else -> "Signal GPS perdu — resynchronisation..."
                                    }
                                    Text(
                                        text = statusText,
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.SemiBold,
                                        color = Color(0xFFE65100)
                                    )
                                }
                            }
                        }

                        AnimatedVisibility(visible = progress.isAlightingSoon) {
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFFFF3E0),
                                modifier = Modifier
                                    .fillMaxWidth()
                                    .padding(top = 8.dp)
                            ) {
                                Row(
                                    verticalAlignment = Alignment.CenterVertically,
                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp)
                                ) {
                                    Icon(
                                        imageVector = Icons.Default.NotificationsActive,
                                        contentDescription = null,
                                        tint = Color(0xFFE65100),
                                        modifier = Modifier.size(18.dp)
                                    )
                                    val stops = progress.stopsRemaining ?: 1
                                    Text(
                                        text = if (stops <= 1) {
                                            "Descente imminente : prochain arrêt !"
                                        } else {
                                            "Descente imminente : dans $stops arrêts"
                                        },
                                        style = MaterialTheme.typography.labelMedium,
                                        fontWeight = FontWeight.Bold,
                                        color = Color(0xFFE65100)
                                    )
                                }
                            }
                        }
                    }
                }

                LazyColumn(
                    state = listState,
                    contentPadding = PaddingValues(top = 12.dp, bottom = 88.dp),
                    modifier = Modifier
                        .fillMaxSize()
                        .navigationBarsPadding()
                ) {
                    val displayLegs = progress.adjustedLegs
                    itemsIndexed(displayLegs) { index, leg ->
                        val legState = when {
                            index < progress.activeLegIndex -> LegNavigationState.PAST
                            index == progress.activeLegIndex -> LegNavigationState.ACTIVE
                            else -> LegNavigationState.UPCOMING
                        }
                        JourneyLegItem(
                            leg = leg,
                            navigationState = legState,
                            activeStopPosition = if (legState == LegNavigationState.ACTIVE) {
                                progress.activeStopPosition
                            } else {
                                null
                            },
                            isAtStop = legState == LegNavigationState.ACTIVE && progress.isAtStop,
                            isAlightingSoon = legState == LegNavigationState.ACTIVE && progress.isAlightingSoon,
                            nowMillis = nowMillis,
                            prevLeg = displayLegs.getOrNull(index - 1),
                            nextLeg = displayLegs.getOrNull(index + 1),
                            isFirstWalk = index == 0,
                            resyncKey = resyncTick + networkReconnectTick,
                            externalArrivalPlatformInfo = liveArrivalPlatforms[index]
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                }
            }

            AnimatedVisibility(
                visible = isUserScrollingAway,
                enter = fadeIn() + slideInVertically { it / 2 },
                exit = fadeOut() + slideOutVertically { it / 2 },
                modifier = Modifier
                    .align(Alignment.BottomCenter)
                    .navigationBarsPadding()
                    .padding(bottom = 20.dp)
                    .zIndex(20f)
            ) {
                Surface(
                    onClick = {
                        isUserScrollingAway = false
                        coroutineScope.launch {
                            scrollToActiveLeg()
                        }
                    },
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.primary,
                    contentColor = MaterialTheme.colorScheme.onPrimary,
                    shadowElevation = 6.dp
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        modifier = Modifier.padding(horizontal = 18.dp, vertical = 12.dp)
                    ) {
                        Icon(
                            imageVector = Icons.Default.MyLocation,
                            contentDescription = "Recentrer",
                            modifier = Modifier.size(18.dp)
                        )
                        Text(
                            text = "Recentrer",
                            fontSize = 14.sp,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                }
            }
        }
    }
}
