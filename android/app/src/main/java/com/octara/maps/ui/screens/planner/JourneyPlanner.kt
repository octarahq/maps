package com.octara.maps.ui.screens.planner

import android.app.DatePickerDialog
import android.app.TimePickerDialog
import androidx.activity.compose.BackHandler
import androidx.compose.animation.core.Spring
import androidx.compose.animation.core.animateDpAsState
import androidx.compose.animation.core.spring
import androidx.compose.foundation.Canvas
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxWithConstraints
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.IntrinsicSize
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.wrapContentHeight
import androidx.compose.foundation.layout.wrapContentSize
import androidx.compose.foundation.layout.wrapContentWidth
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AccessTime
import androidx.compose.material.icons.automirrored.filled.Accessible
import androidx.compose.material.icons.automirrored.filled.DirectionsBike
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.SwapVert
import android.content.Context
import androidx.compose.material.icons.automirrored.filled.DirectionsWalk
import androidx.compose.material.icons.filled.FlashOn
import androidx.compose.material.icons.filled.NearMe
import androidx.compose.material.icons.filled.Weekend
import androidx.compose.material.icons.filled.ExpandLess
import androidx.compose.material.icons.filled.ExpandMore
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.DirectionsBus
import androidx.compose.material.icons.filled.Subway
import androidx.compose.material.icons.filled.Train
import androidx.compose.material.icons.filled.Tram
import androidx.compose.material.icons.filled.DirectionsBoat
import androidx.compose.material.icons.filled.Cable
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.filled.WarningAmber
import androidx.compose.material.icons.filled.SearchOff
import androidx.compose.material.icons.filled.LocationOff
import androidx.compose.material.icons.filled.MyLocation
import androidx.compose.material.icons.filled.Timer
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Button
import androidx.compose.material3.HorizontalDivider
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SheetState
import androidx.compose.material3.SheetValue
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import com.octara.maps.ui.theme.AppDimensions
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.geometry.CornerRadius
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.RoundRect
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.Path
import androidx.compose.ui.input.nestedscroll.NestedScrollConnection
import androidx.compose.ui.input.nestedscroll.NestedScrollSource
import androidx.compose.ui.input.nestedscroll.nestedScroll
import androidx.compose.ui.platform.LocalConfiguration
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.octara.maps.navigation.LocalRouter
import com.octara.maps.navigation.Screen
import com.octara.maps.types.JourneyQuery
import com.octara.maps.types.JourneyResponse
import com.octara.maps.types.ParsedTrip
import com.octara.maps.types.TripLeg
import com.octara.maps.types.Position
import com.octara.maps.types.TimeType
import com.octara.maps.types.Route
import com.octara.maps.types.Accessibility
import com.octara.maps.types.RouteType
import com.octara.maps.types.TrainPosition
import com.octara.maps.types.TripPositionInTrain
import com.octara.maps.utils.LocationHelper
import com.octara.maps.utils.LocationManager
import com.octara.maps.utils.api
import com.octara.maps.ui.components.JourneyLegItem
import com.octara.maps.ui.components.RouteIcon
import com.octara.maps.ui.components.PositionPicker
import com.octara.maps.ui.components.bottomSheetContentScroll
import com.octara.maps.ui.components.formatTimeString
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.draw.drawWithContent
import androidx.compose.ui.graphics.Brush
import androidx.compose.ui.graphics.BlendMode
import androidx.compose.ui.graphics.graphicsLayer
import coil.compose.AsyncImage
import kotlinx.coroutines.launch
import java.text.SimpleDateFormat
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter
import java.util.Calendar
import java.util.Locale

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun JourneyPlannerSheetContent(
    toStopId: String = "",
    toStopName: String,
    toLat: Double? = null,
    toLon: Double? = null,
    modifier: Modifier = Modifier,
    sheetState: SheetState? = null,
    onClose: (() -> Unit)? = null
) {
    val router = LocalRouter.current
    val configuration = LocalConfiguration.current
    val context = LocalContext.current
    val prefs = remember { context.getSharedPreferences("journey_planner_prefs", Context.MODE_PRIVATE) }

    val allRouteModes = remember {
        listOf(
            RouteType.METRO,
            RouteType.TRAIN,
            RouteType.TRAM,
            RouteType.BUS,
            RouteType.FERRY,
            RouteType.CABLE_TRAM
        )
    }

    val initialSavedModes = remember {
        val saved = prefs.getStringSet("preferred_allowed_modes", null)
        if (saved == null) {
            allRouteModes.toSet()
        } else {
            saved.mapNotNull { name ->
                try {
                    RouteType.valueOf(name)
                } catch (_: Exception) {
                    null
                }
            }.toSet().ifEmpty { allRouteModes.toSet() }
        }
    }

    var selectedAllowedModes by remember { mutableStateOf(initialSavedModes) }
    val gpsLocation by LocationManager.currentLocation.collectAsState()
    val coroutineScope = rememberCoroutineScope()

    val initialWheelchair = remember { prefs.getBoolean("preferred_wheelchair", false) }
    val initialWithBike = remember { prefs.getBoolean("preferred_with_bike", false) }

    var journeyConfig by remember {
        mutableStateOf(
            JourneyQuery(
                from = Position.CurrentLocation,
                to = if (toLat != null && toLon != null) {
                    Position.Coordinates(toLat, toLon)
                } else {
                    Position.ByStop(toStopId)
                },
                time = LocalDateTime.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd'T'HH:mm:ss")),
                timeType = TimeType.DEPARTURE,
                allowedModes = if (selectedAllowedModes.size == allRouteModes.size) null else selectedAllowedModes.toList(),
                withBike = initialWithBike,
                wheelchair = initialWheelchair,
                currentGps = null
            )
        )
    }

    LaunchedEffect(Unit) {
        LocationManager.fetchCurrentLocation(context)
    }

    LaunchedEffect(gpsLocation) {
        val gps = gpsLocation
        if (gps != null) {
            val coords = Pair(gps.first.toDouble(), gps.second.toDouble())
            val prev = journeyConfig.currentGps
            val shouldUpdate = if (prev == null) {
                true
            } else {
                val res = FloatArray(1)
                android.location.Location.distanceBetween(prev.first, prev.second, coords.first, coords.second, res)
                res[0] > 120f
            }
            if (shouldUpdate) {
                journeyConfig = journeyConfig.copy(currentGps = coords)
            }
        } else {
            journeyConfig = journeyConfig.copy(currentGps = null)
        }
    }

    var journeys by remember { mutableStateOf(JourneyResponse()) }
    var isLoading by remember { mutableStateOf(false) }

    val needsGps = (journeyConfig.from is Position.CurrentLocation) || (journeyConfig.to is Position.CurrentLocation)
    val isGpsMissing = needsGps && journeyConfig.currentGps == null

    LaunchedEffect(journeyConfig) {
        if (needsGps && journeyConfig.currentGps == null) {
            journeys = JourneyResponse()
            isLoading = false
            return@LaunchedEffect
        }
        isLoading = true
        journeys = JourneyResponse()
        try {
            journeys = api.getJourney(query = journeyConfig)
        } catch (_: Exception) {
        } finally {
            isLoading = false
        }
    }

    var currentFromStopName by remember { mutableStateOf("Ma position") }
    var currentToStopName by remember { mutableStateOf(toStopName) }
    var activePickerTarget by remember { mutableStateOf<String?>(null) }

    var selectedCalendar by remember { mutableStateOf(Calendar.getInstance()) }
    var isMenuExpanded by remember { mutableStateOf(false) }
    var showSettingsModal by remember { mutableStateOf(false) }

    fun formatDisplayDateTime(cal: Calendar): String {
        val now = Calendar.getInstance()
        val isToday = now.get(Calendar.YEAR) == cal.get(Calendar.YEAR) &&
                now.get(Calendar.DAY_OF_YEAR) == cal.get(Calendar.DAY_OF_YEAR)
        val isTomorrow = now.get(Calendar.YEAR) == cal.get(Calendar.YEAR) &&
                now.get(Calendar.DAY_OF_YEAR) + 1 == cal.get(Calendar.DAY_OF_YEAR)
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        val minute = String.format(Locale.getDefault(), "%02d", cal.get(Calendar.MINUTE))
        return when {
            isToday -> "${hour}h$minute"
            isTomorrow -> "Demain ${hour}h$minute"
            else -> "${cal.get(Calendar.DAY_OF_MONTH)}/${cal.get(Calendar.MONTH) + 1} ${hour}h$minute"
        }
    }

    val openDateTimePicker: () -> Unit = {
        val datePicker = DatePickerDialog(
            context,
            { _, year, month, dayOfMonth ->
                val timePicker = TimePickerDialog(
                    context,
                    { _, hourOfDay, minute ->
                        val chosen = Calendar.getInstance().apply {
                            set(Calendar.YEAR, year)
                            set(Calendar.MONTH, month)
                            set(Calendar.DAY_OF_MONTH, dayOfMonth)
                            set(Calendar.HOUR_OF_DAY, hourOfDay)
                            set(Calendar.MINUTE, minute)
                            set(Calendar.SECOND, 0)
                        }
                        val now = Calendar.getInstance()
                        val finalCal = if (chosen.before(now)) now else chosen
                        selectedCalendar = finalCal
                        val isoFormat = SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss", Locale.getDefault())
                        journeyConfig = journeyConfig.copy(time = isoFormat.format(finalCal.time))
                    },
                    selectedCalendar.get(Calendar.HOUR_OF_DAY),
                    selectedCalendar.get(Calendar.MINUTE),
                    true
                )
                timePicker.show()
            },
            selectedCalendar.get(Calendar.YEAR),
            selectedCalendar.get(Calendar.MONTH),
            selectedCalendar.get(Calendar.DAY_OF_MONTH)
        )
        datePicker.datePicker.minDate = System.currentTimeMillis() - 1000L
        datePicker.show()
    }

    var isFullscreen by remember { mutableStateOf(false) }

    BackHandler(enabled = isFullscreen) {
        isFullscreen = false
    }

    if (sheetState != null) {
        LaunchedEffect(sheetState.currentValue) {
            if (sheetState.currentValue == SheetValue.Hidden) {
                onClose?.invoke() ?: router.goBack()
            } else if (sheetState.currentValue == SheetValue.PartiallyExpanded) {
                isFullscreen = false
            }
        }
    }

    LaunchedEffect(Unit) {
        if (sheetState != null && sheetState.currentValue != SheetValue.Expanded) {
            try {
                sheetState.expand()
            } catch (_: Exception) {
            }
        }
    }

    val scrollState = rememberScrollState()

    val screenHeight = configuration.screenHeightDp.dp
    val targetHeight = if (isFullscreen) screenHeight else (screenHeight * 0.82f)
    val animatedHeight by animateDpAsState(
        targetValue = targetHeight,
        animationSpec = spring(
            dampingRatio = Spring.DampingRatioNoBouncy,
            stiffness = Spring.StiffnessMediumLow
        ),
        label = "journeyPlannerHeight"
    )

    Column(
        modifier = modifier
            .fillMaxWidth()
            .height(animatedHeight)
            .then(if (isFullscreen) Modifier.statusBarsPadding() else Modifier)
            .navigationBarsPadding()
            .bottomSheetContentScroll(
                scrollState = scrollState,
                sheetState = sheetState,
                onBoundaryDragUp = {
                    if (!isFullscreen && sheetState?.currentValue == SheetValue.Expanded) {
                        isFullscreen = true
                    }
                },
                onBoundaryDragDown = if (isFullscreen) {
                    { isFullscreen = false }
                } else {
                    null
                }
            )
            .verticalScroll(scrollState)
            .padding(horizontal = 16.dp, vertical = 8.dp)
    ) {
        val fromIsCurrentLocation = journeyConfig.from is Position.CurrentLocation
        val fromText = if (fromIsCurrentLocation) {
            if (gpsLocation != null) "Ma position" else "Ma position (introuvable)"
        } else currentFromStopName
        val fromColor = if (fromIsCurrentLocation) {
            if (gpsLocation != null) Color(0xFF007AFF) else MaterialTheme.colorScheme.error
        } else Color(0xFF34C759)

        val toIsCurrentLocation = journeyConfig.to is Position.CurrentLocation
        val toText = if (toIsCurrentLocation) {
            if (gpsLocation != null) "Ma position" else "Ma position (introuvable)"
        } else currentToStopName
        val toColor = if (toIsCurrentLocation) {
            if (gpsLocation != null) Color(0xFF007AFF) else MaterialTheme.colorScheme.error
        } else Color(0xFF34C759)

        Surface(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(12.dp),
            color = Color(0xFFE8E8ED)
        ) {
            Box(
                modifier = Modifier.padding(horizontal = 14.dp, vertical = 6.dp),
                contentAlignment = Alignment.CenterEnd
            ) {
                Column(
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .clickable {
                                activePickerTarget = "from"
                            }
                            .padding(vertical = 10.dp, horizontal = 4.dp)
                    ) {
                        Text(
                            text = fromText,
                            color = Color.Black,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.padding(end = 48.dp)
                        )
                    }
                    HorizontalDivider(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(end = 40.dp),
                        thickness = 1.dp,
                        color = Color(0xFFE0E0E0)
                    )
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier
                            .fillMaxWidth()
                            .clip(RoundedCornerShape(8.dp))
                            .clickable {
                                activePickerTarget = "to"
                            }
                            .padding(vertical = 10.dp, horizontal = 4.dp)
                    ) {
                        Text(
                            text = toText,
                            color = Color.Black,
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1,
                            overflow = TextOverflow.Ellipsis,
                            modifier = Modifier.padding(end = 48.dp)
                        )
                    }
                }

                Surface(
                    shape = RoundedCornerShape(percent = 50),
                    color = Color.White,
                    shadowElevation = 2.dp,
                    modifier = Modifier
                        .width(48.dp)
                        .height(32.dp)
                        .clip(RoundedCornerShape(percent = 50))
                        .clickable {
                            val prevFrom = journeyConfig.from
                            val prevTo = journeyConfig.to
                            val prevFromName = currentFromStopName
                            val prevToName = currentToStopName
                            currentFromStopName = prevToName
                            currentToStopName = prevFromName
                            journeyConfig = journeyConfig.copy(
                                from = prevTo,
                                to = prevFrom
                            )
                        }
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Icon(
                            imageVector = Icons.Default.SwapVert,
                            contentDescription = "Inverser",
                            tint = Color(0xFF555555),
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .height(IntrinsicSize.Min)
                .padding(top = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Surface(
                modifier = Modifier
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { isMenuExpanded = true },
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFFE8E8ED)
            ) {
                Box(
                    modifier = Modifier
                        .padding(horizontal = 12.dp, vertical = 10.dp),
                    contentAlignment = Alignment.CenterStart
                ) {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(6.dp),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = if (journeyConfig.timeType == TimeType.DEPARTURE) "Départ à" else "Arrivée à",
                            color = Color(0xFF555555),
                            fontSize = 15.sp,
                            fontWeight = FontWeight.Bold,
                            maxLines = 1
                        )
                        Icon(
                            imageVector = Icons.Default.ArrowDropDown,
                            contentDescription = null,
                            tint = Color(0xFF555555),
                            modifier = Modifier.size(20.dp)
                        )
                    }

                    DropdownMenu(
                        expanded = isMenuExpanded,
                        onDismissRequest = { isMenuExpanded = false }
                    ) {
                        DropdownMenuItem(
                            text = { Text("Départ à", fontWeight = FontWeight.SemiBold) },
                            onClick = {
                                journeyConfig = journeyConfig.copy(timeType = TimeType.DEPARTURE)
                                isMenuExpanded = false
                            }
                        )
                        DropdownMenuItem(
                            text = { Text("Arrivée à", fontWeight = FontWeight.SemiBold) },
                            onClick = {
                                journeyConfig = journeyConfig.copy(timeType = TimeType.ARRIVAL)
                                isMenuExpanded = false
                            }
                        )
                    }
                }
            }

            Surface(
                modifier = Modifier
                    .weight(1f)
                    .fillMaxHeight()
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { openDateTimePicker() },
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFFE8E8ED)
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(horizontal = 14.dp, vertical = 10.dp),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = formatDisplayDateTime(selectedCalendar),
                        color = Color(0xFF555555),
                        fontSize = 15.sp,
                        fontWeight = FontWeight.Bold
                    )
                    Icon(
                        imageVector = Icons.Default.AccessTime,
                        contentDescription = "Choisir l'heure",
                        tint = Color(0xFF555555),
                        modifier = Modifier.size(18.dp)
                    )
                }
            }

            Surface(
                modifier = Modifier
                    .fillMaxHeight()
                    .aspectRatio(1f)
                    .clip(RoundedCornerShape(12.dp))
                    .clickable { showSettingsModal = true },
                shape = RoundedCornerShape(12.dp),
                color = Color(0xFFE8E8ED)
            ) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center
                ) {
                    Icon(
                        imageVector = Icons.Default.MoreVert,
                        contentDescription = "Options",
                        tint = Color(0xFF555555),
                        modifier = Modifier.size(20.dp)
                    )
                }
            }
        }

        if (showSettingsModal) {
            Dialog(
                onDismissRequest = { showSettingsModal = false },
                properties = DialogProperties(usePlatformDefaultWidth = false)
            ) {
                Surface(
                    modifier = Modifier
                        .fillMaxWidth(0.9f)
                        .clip(RoundedCornerShape(24.dp)),
                    shape = RoundedCornerShape(24.dp),
                    color = Color.White,
                    shadowElevation = 8.dp
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(24.dp)
                    ) {
                        Text(
                            text = "Paramètres divers",
                            fontSize = 22.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color(0xFF1E1E1E)
                        )

                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 16.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Accessibilité fauteuil roulant",
                                fontSize = 16.sp,
                                fontWeight = FontWeight.Medium,
                                color = Color(0xFF1E1E1E)
                            )
                            androidx.compose.material3.Switch(
                                checked = journeyConfig.wheelchair,
                                onCheckedChange = { isChecked ->
                                    journeyConfig = journeyConfig.copy(wheelchair = isChecked)
                                    prefs.edit().putBoolean("preferred_wheelchair", isChecked).apply()
                                },
                                colors = androidx.compose.material3.SwitchDefaults.colors(
                                    checkedThumbColor = Color(0xFF007AFF),
                                    uncheckedThumbColor = Color(0xFFBDBDBD),
                                    checkedTrackColor = Color(0xFFB3D7FF),
                                    uncheckedTrackColor = Color(0xFFE0E0E0)
                                )
                            )
                        }

                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(top = 16.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Départ avec vélo",
                                fontSize = 16.sp,
                                fontWeight = FontWeight.Medium,
                                color = Color(0xFF1E1E1E)
                            )
                            androidx.compose.material3.Switch(
                                checked = journeyConfig.withBike,
                                onCheckedChange = { isChecked ->
                                    journeyConfig = journeyConfig.copy(withBike = isChecked)
                                    prefs.edit().putBoolean("preferred_with_bike", isChecked).apply()
                                },
                                colors = androidx.compose.material3.SwitchDefaults.colors(
                                    checkedThumbColor = Color(0xFF007AFF),
                                    uncheckedThumbColor = Color(0xFFBDBDBD),
                                    checkedTrackColor = Color(0xFFB3D7FF),
                                    uncheckedTrackColor = Color(0xFFE0E0E0)
                                )
                            )
                        }

                        HorizontalDivider(
                            modifier = Modifier.padding(vertical = 18.dp),
                            color = Color(0xFFEBEBEB)
                        )

                        Text(
                            text = "Modes de transport autorisés",
                            fontSize = 17.sp,
                            fontWeight = FontWeight.Bold,
                            color = Color(0xFF1E1E1E),
                            modifier = Modifier.padding(bottom = 12.dp)
                        )

                        val modeMeta = remember {
                            listOf(
                                RouteType.METRO to ("Métro" to Icons.Filled.Subway),
                                RouteType.TRAIN to ("Train" to Icons.Filled.Train),
                                RouteType.TRAM to ("Tramway" to Icons.Filled.Tram),
                                RouteType.BUS to ("Bus" to Icons.Filled.DirectionsBus),
                                RouteType.FERRY to ("Ferry" to Icons.Filled.DirectionsBoat),
                                RouteType.CABLE_TRAM to ("Téléphérique" to Icons.Filled.Cable)
                            )
                        }

                        Column(
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            for (rowModes in modeMeta.chunked(2)) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                                ) {
                                    for ((mode, info) in rowModes) {
                                        val isSelected = selectedAllowedModes.contains(mode)
                                        Surface(
                                            modifier = Modifier
                                                .weight(1f)
                                                .clip(RoundedCornerShape(12.dp))
                                                .clickable {
                                                    val newSet = if (isSelected) {
                                                        if (selectedAllowedModes.size > 1) selectedAllowedModes - mode else selectedAllowedModes
                                                    } else {
                                                        selectedAllowedModes + mode
                                                    }
                                                    selectedAllowedModes = newSet
                                                    prefs.edit()
                                                        .putStringSet(
                                                            "preferred_allowed_modes",
                                                            newSet.map { it.name }.toSet()
                                                        )
                                                        .apply()
                                                    journeyConfig = journeyConfig.copy(
                                                        allowedModes = if (newSet.size == allRouteModes.size) null else newSet.toList()
                                                    )
                                                },
                                            shape = RoundedCornerShape(12.dp),
                                            color = if (isSelected) Color(0xFFEBF3FF) else Color(0xFFF3F3F5),
                                            border = if (isSelected) androidx.compose.foundation.BorderStroke(
                                                1.dp,
                                                Color(0xFF007AFF)
                                            ) else null
                                        ) {
                                            Row(
                                                modifier = Modifier
                                                    .fillMaxWidth()
                                                    .padding(horizontal = 12.dp, vertical = 10.dp),
                                                horizontalArrangement = Arrangement.SpaceBetween,
                                                verticalAlignment = Alignment.CenterVertically
                                            ) {
                                                Row(
                                                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                                                    verticalAlignment = Alignment.CenterVertically,
                                                    modifier = Modifier.weight(1f, fill = false)
                                                ) {
                                                    Icon(
                                                        imageVector = info.second,
                                                        contentDescription = null,
                                                        tint = if (isSelected) Color(0xFF007AFF) else Color(0xFF666666),
                                                        modifier = Modifier.size(20.dp)
                                                    )
                                                    Text(
                                                        text = info.first,
                                                        fontSize = 14.sp,
                                                        fontWeight = if (isSelected) FontWeight.SemiBold else FontWeight.Medium,
                                                        color = if (isSelected) Color(0xFF007AFF) else Color(0xFF444444),
                                                        maxLines = 1,
                                                        overflow = androidx.compose.ui.text.style.TextOverflow.Ellipsis
                                                    )
                                                }
                                                if (isSelected) {
                                                    Icon(
                                                        imageVector = Icons.Default.Check,
                                                        contentDescription = "Sélectionné",
                                                        tint = Color(0xFF007AFF),
                                                        modifier = Modifier.size(16.dp)
                                                    )
                                                }
                                            }
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

        var selectedMode by remember {
            mutableStateOf(prefs.getString("preferred_route_mode", "comfort") ?: "comfort")
        }

        data class RouteModeOption(
            val key: String,
            val label: String,
            val icon: androidx.compose.ui.graphics.vector.ImageVector
        )

        val routeModes = remember {
            listOf(
                RouteModeOption("comfort", "Confort", Icons.Default.Weekend),
                RouteModeOption("fastest", "Rapide", Icons.Default.FlashOn),
                RouteModeOption("least_walk", "Moins de marche", Icons.AutoMirrored.Filled.DirectionsWalk),
                RouteModeOption("nearest_station", "Proche", Icons.Default.NearMe)
            )
        }

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            horizontalArrangement = Arrangement.spacedBy(6.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            for (mode in routeModes) {
                val isSelected = (mode.key == selectedMode)
                val trip = journeys.journeys[mode.key]
                val timeLabel = when {
                    trip == null -> ""
                    trip.durationSec < 3600 -> "${trip.durationSec / 60}m"
                    else -> "${trip.durationSec / 3600}h${(trip.durationSec % 3600) / 60}"
                }

                Surface(
                    modifier = Modifier
                        .weight(if (isSelected) 1.6f else 1f)
                        .clip(RoundedCornerShape(12.dp))
                        .clickable {
                            selectedMode = mode.key
                            prefs.edit().putString("preferred_route_mode", mode.key).apply()
                        },
                    shape = RoundedCornerShape(12.dp),
                    color = if (isSelected) MaterialTheme.colorScheme.primaryContainer else Color(0xFFE8E8ED)
                ) {
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(horizontal = 6.dp, vertical = 8.dp),
                        horizontalArrangement = Arrangement.Center,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        if (isSelected) {
                            Text(
                                text = mode.label,
                                fontSize = 12.sp,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.onPrimaryContainer,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis
                            )
                        } else {
                            Icon(
                                imageVector = mode.icon,
                                contentDescription = mode.label,
                                modifier = Modifier.size(16.dp),
                                tint = Color(0xFF555555)
                            )
                            if (timeLabel.isNotEmpty()) {
                                Spacer(modifier = Modifier.width(3.dp))
                                Text(
                                    text = timeLabel,
                                    fontSize = 11.sp,
                                    fontWeight = FontWeight.SemiBold,
                                    color = Color(0xFF555555),
                                    maxLines = 1
                                )
                            }
                        }
                    }
                }
            }
        }

        HorizontalDivider(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 10.dp),
            thickness = 1.dp,
            color = Color(0xFFE0E0E0)
        )

        val journey =
            journeys.journeys[selectedMode] ?: journeys.journeys["comfort"] ?: journeys.journeys.values.firstOrNull()

        LaunchedEffect(journey) {
            router.activeJourney = journey
        }

        if (isGpsMissing) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 32.dp),
                contentAlignment = Alignment.Center
            ) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(20.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(10.dp)
                    ) {
                        Box(
                            modifier = Modifier
                                .size(48.dp)
                                .clip(RoundedCornerShape(14.dp))
                                .background(MaterialTheme.colorScheme.errorContainer.copy(alpha = 0.5f)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = Icons.Default.LocationOff,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.error,
                                modifier = Modifier.size(26.dp)
                            )
                        }
                        Text(
                            text = "Position GPS introuvable",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = "Impossible de récupérer votre position actuelle. Activez la localisation ou sélectionnez manuellement un point de départ.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            textAlign = androidx.compose.ui.text.style.TextAlign.Center
                        )
                        Button(
                            onClick = {
                                coroutineScope.launch {
                                    LocationManager.fetchCurrentLocation(context, forceRefresh = true)
                                }
                            },
                            shape = RoundedCornerShape(10.dp),
                            colors = androidx.compose.material3.ButtonDefaults.buttonColors(
                                containerColor = MaterialTheme.colorScheme.primaryContainer,
                                contentColor = MaterialTheme.colorScheme.onPrimaryContainer
                            )
                        ) {
                            Icon(
                                imageVector = Icons.Default.MyLocation,
                                contentDescription = null,
                                modifier = Modifier.size(16.dp)
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Text(
                                text = "Réessayer la localisation",
                                style = MaterialTheme.typography.labelMedium,
                                fontWeight = FontWeight.Bold
                            )
                        }
                    }
                }
            }
        } else if (isLoading) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 36.dp),
                contentAlignment = Alignment.Center
            ) {
                Column(
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(32.dp),
                        color = MaterialTheme.colorScheme.primary,
                        strokeWidth = 3.dp
                    )
                    Text(
                        text = "Recherche du meilleur itinéraire...",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        } else if (journey == null) {
            Box(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 32.dp),
                contentAlignment = Alignment.Center
            ) {
                Surface(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(16.dp),
                    color = MaterialTheme.colorScheme.surfaceVariant.copy(alpha = 0.4f)
                ) {
                    Column(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(20.dp),
                        horizontalAlignment = Alignment.CenterHorizontally,
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Box(
                            modifier = Modifier
                                .size(48.dp)
                                .clip(RoundedCornerShape(14.dp)),
                            contentAlignment = Alignment.Center
                        ) {
                            Icon(
                                imageVector = if (journeys.journeys.isEmpty()) Icons.Default.SearchOff else Icons.Default.WarningAmber,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.size(26.dp)
                            )
                        }
                        Text(
                            text = if (journeys.journeys.isEmpty()) "Aucun itinéraire trouvé" else "Option non disponible",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                        Text(
                            text = if (journeys.journeys.isEmpty()) {
                                "Essayez de modifier vos critères ou les modes de transport autorisés."
                            } else {
                                "Ce mode de trajet n'a pas donné de résultat pour cet itinéraire."
                            },
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant,
                            textAlign = androidx.compose.ui.text.style.TextAlign.Center
                        )
                    }
                }
            }
        } else {
            val durationText = when {
                journey.durationSec < 3600 -> "${journey.durationSec / 60} min"
                else -> {
                    val h = journey.durationSec / 3600
                    val m = (journey.durationSec % 3600) / 60
                    if (m == 0) "${h}h" else "${h}h ${m}min"
                }
            }

            val depTimeStr = formatTimeString(journey.departureTime)
            val arrTimeStr = formatTimeString(journey.arrivalTime)

            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 14.dp, bottom = 8.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.Bottom
            ) {
                Column(verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(
                        text = "Durée estimée",
                        style = MaterialTheme.typography.labelSmall,
                        fontWeight = FontWeight.Medium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Text(
                            text = durationText,
                            style = MaterialTheme.typography.headlineSmall,
                            fontWeight = FontWeight.ExtraBold,
                            color = MaterialTheme.colorScheme.onSurface
                        )
                    }
                }

                if (depTimeStr.isNotBlank() && arrTimeStr.isNotBlank()) {
                    Column(
                        horizontalAlignment = Alignment.End,
                        verticalArrangement = Arrangement.spacedBy(2.dp)
                    ) {
                        Text(
                            text = "Horaires",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Medium,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Text(
                            text = "$depTimeStr → $arrTimeStr",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Bold,
                            color = MaterialTheme.colorScheme.primary
                        )
                    }
                }
            }

            if (journeyConfig.from == Position.CurrentLocation) {
                Button(
                    onClick = {
                        router.navigateTo(Screen.Navigation(trip = journey))
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 4.dp, bottom = 12.dp),
                    shape = RoundedCornerShape(12.dp),
                    colors = androidx.compose.material3.ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.primaryContainer,
                        contentColor = MaterialTheme.colorScheme.onPrimaryContainer
                    )
                ) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.Center,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "Démarrer l'itinéraire",
                            style = MaterialTheme.typography.labelLarge,
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(end = 8.dp)
                        )
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                            contentDescription = "Itinéraire",
                            modifier = Modifier.size(18.dp)
                        )
                    }
                }
            }

            journey.legs.forEachIndexed { index, leg ->
                JourneyLegItem(
                    leg = leg,
                    prevLeg = journey.legs.getOrNull(index - 1),
                    nextLeg = journey.legs.getOrNull(index + 1),
                    isFirstWalk = index == 0
                )
                Spacer(modifier = Modifier.height(8.dp))
            }
        }
    }

    if (activePickerTarget != null) {
        Dialog(
            onDismissRequest = { activePickerTarget = null },
            properties = DialogProperties(
                usePlatformDefaultWidth = false,
                decorFitsSystemWindows = false
            )
        ) {
            PositionPicker(
                onDismiss = { activePickerTarget = null },
                onSelectPosition = { pos, name ->
                    if (activePickerTarget == "from") {
                        currentFromStopName = name
                        journeyConfig = journeyConfig.copy(from = pos)
                    } else if (activePickerTarget == "to") {
                        currentToStopName = name
                        journeyConfig = journeyConfig.copy(to = pos)
                    }
                    activePickerTarget = null
                }
            )
        }
    }
}
